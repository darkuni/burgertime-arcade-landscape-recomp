package sound

import "sync"

// ayVolume is the AY-3-8910 amplitude curve (4-bit level -> linear gain).
var ayVolume = [16]float64{
	0, 0.0100, 0.0145, 0.0211, 0.0307, 0.0455, 0.0644, 0.1073,
	0.1288, 0.2047, 0.2913, 0.3722, 0.4921, 0.6349, 0.8175, 1.0,
}

// ay is one AY-3-8910, stepped at its tone clock (clock / 16).
type ay struct {
	tone      [3]int
	level     [3]bool
	noiseC    int
	noise     uint32
	noiseHi   bool
	envC      int
	envPos    int
	envHold   bool
	envLvl    int
	envUp     bool
	envShape  byte
	lastShape byte
}

func newAY() *ay { return &ay{noise: 1, envLvl: 0} }

// restartEnv is called when the envelope shape register is written.
func (a *ay) restartEnv(shape byte) {
	a.envShape = shape & 15
	a.envPos = 0
	a.envHold = false
	if shape&4 != 0 { // attack: count up from 0
		a.envUp, a.envLvl = true, 0
	} else {
		a.envUp, a.envLvl = false, 15
	}
	a.envC = 0
}

func (a *ay) stepEnv() {
	if a.envHold {
		return
	}
	if a.envUp {
		a.envLvl++
	} else {
		a.envLvl--
	}
	if a.envLvl >= 0 && a.envLvl <= 15 {
		return
	}
	sh := a.envShape
	switch {
	case sh&8 == 0: // a single ramp, then silence
		a.envLvl, a.envHold = 0, true
	case sh&1 != 0: // hold the final level
		a.envHold = true
		end := a.envUp
		if sh&2 != 0 {
			end = !end
		}
		if end {
			a.envLvl = 15
		} else {
			a.envLvl = 0
		}
	case sh&2 != 0: // alternate: turn round
		a.envUp = !a.envUp
		if a.envUp {
			a.envLvl = 0
		} else {
			a.envLvl = 15
		}
	default: // repeat the ramp
		if a.envUp {
			a.envLvl = 0
		} else {
			a.envLvl = 15
		}
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// step advances the chip by one tone-clock tick and returns the sum of its three outputs.
func (a *ay) step(r *[16]byte) float64 {
	for c := 0; c < 3; c++ {
		p := int(r[2*c]) | int(r[2*c+1]&15)<<8
		if p < 1 {
			p = 1
		}
		a.tone[c]--
		if a.tone[c] <= 0 {
			a.tone[c] += p
			a.level[c] = !a.level[c]
		}
	}
	np := int(r[6]&31) * 2
	if np < 2 {
		np = 2
	}
	a.noiseC--
	if a.noiseC <= 0 {
		a.noiseC += np
		bit := (a.noise ^ a.noise>>3) & 1
		a.noise = a.noise>>1 | bit<<16
		a.noiseHi = a.noise&1 != 0
	}
	ep := int(r[11]) | int(r[12])<<8
	if ep < 1 {
		ep = 1
	}
	a.envC++
	if a.envC >= ep*16 {
		a.envC = 0
		a.stepEnv()
	}
	var sum float64
	for c := 0; c < 3; c++ {
		toneOn := a.level[c] || r[7]>>uint(c)&1 != 0
		noiseOn := a.noiseHi || r[7]>>uint(3+c)&1 != 0
		if !(toneOn && noiseOn) {
			continue
		}
		v := int(r[8+c] & 15)
		if r[8+c]&0x10 != 0 {
			v = a.envLvl
		}
		sum += ayVolume[v]
	}
	return sum
}

// Synth renders the board's sound as audio samples.
type Synth struct {
	mu       sync.Mutex
	seq      *Seq
	chips    [2]*ay
	rate     float64
	cycAcc   float64 // sound cycles owed to the sequencer
	tickAcc  float64
	hpX, hpY float64
	pend     []byte
}

// NewSynth creates a synthesiser for the given sound ROM and output rate.
func NewSynth(rom []byte, rate int) *Synth {
	s := &Synth{seq: New(rom), rate: float64(rate)}
	s.chips[0], s.chips[1] = newAY(), newAY()
	s.seq.OnAY = func(w AYWrite) {
		if w.Reg == 13 {
			s.chips[w.Chip].restartEnv(w.Val)
		}
	}
	return s
}

// Command forwards a command byte from the main board.
func (s *Synth) Command(v byte) {
	s.mu.Lock()
	s.pend = append(s.pend, v)
	s.mu.Unlock()
}

// Render fills out with mono samples in [-1, 1].
func (s *Synth) Render(out []float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.pend {
		s.seq.Command(c)
	}
	s.pend = s.pend[:0]
	cycPer := float64(ClockHz) / s.rate
	ticksPer := float64(AYClockHz) / 16 / s.rate
	for i := range out {
		s.cycAcc += cycPer
		n := int(s.cycAcc)
		s.cycAcc -= float64(n)
		s.seq.Run(n)
		s.tickAcc += ticksPer
		k := int(s.tickAcc)
		s.tickAcc -= float64(k)
		var mix float64
		if k > 0 {
			for t := 0; t < k; t++ {
				mix += s.chips[0].step(&s.seq.AY[0]) + s.chips[1].step(&s.seq.AY[1])
			}
			mix /= float64(k)
		}
		mix *= 0.4
		y := mix - s.hpX + 0.995*s.hpY
		s.hpX, s.hpY = mix, y
		if y > 1 {
			y = 1
		} else if y < -1 {
			y = -1
		}
		out[i] = float32(y)
	}
}
