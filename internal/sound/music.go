package sound

// The music engine: the program's timer interrupt handler (f07a). Each interrupt
// runs one of five phases: the note of voice A, B, C, the three duration
// counters, and an idle pass. Voices play on the first chip.

// Zero page variables of the music engine.
const (
	zEnable   = 0x01 // non zero while music plays
	zSaved    = 0x02 // $01 saved while a command is handled
	zTune     = 0x03 // 0 / $ff: which tune restarts when the music ends
	zPhase    = 0x04
	zPtr      = 0x06 // $06/07, $08/09, $0a/0b: note streams of the three voices
	zIdx      = 0x0c // $0c..$0e: stream positions
	zDur      = 0x0f // $0f..$11: duration counters
	zHold     = 0x12 // $12..$14: note is being held
	zEnd      = 0x15
	zPeriodLo = 0x1a
	zPeriodHi = 0x1b
	zVol      = 0x1e // $1e..$20: volumes of the three voices
	zCmd      = 0x3d
	zPrio     = 0x3e
	zBusyTune = 0x43 // non zero while a jingle plays: the tune commands are ignored
)

// nmi is the timer interrupt; the cycles it uses are taken from the main program.
func (s *Seq) nmi() {
	z := &s.z
	c := 10 + 13 + 3 // vector and JMP, pushes, LDA $01
	if z[zEnable] == 0 {
		s.busy += c + 3 + 22
		return
	}
	c += 2 + 3 + 2 // BEQ, LDA $04, CMP #5
	a := z[zPhase]
	old := a
	if a == 5 {
		a = 0
		z[zPhase] = 0
		c += 2 + 2 + 3
	} else {
		c += 3
	}
	z[zPhase]++
	c += 5 + 2 + 2 + 4 + 3 + 4 + 3 + 5
	switch a {
	case 0, 1, 2:
		c += s.voiceNote(int(a), c)
	case 3:
		c += s.durations()
	default:
	}
	s.busy += c + 22 // the exit (f074)
	if OnNMI != nil {
		OnNMI(old, c+22)
	}
}

// OnNMI, if set, is told the cost of every timer interrupt (testing).
var OnNMI func(phase byte, cycles int)

// stream reads a byte of voice v's note stream.
func (s *Seq) stream(v int, idx byte) byte {
	p := uint16(s.z[zPtr+2*v]) | uint16(s.z[zPtr+2*v+1])<<8
	return s.rb(p + uint16(idx))
}

// voiceNote starts the next note of voice v unless the previous one is still held.
// at is the handler's cycle count so far (for the time stamps of the register writes).
func (s *Seq) voiceNote(v int, at int) (c int) {
	z := &s.z
	silent, rest := false, false
	if v == 0 {
		c = 3 // LDA $12
		if z[zHold] != 0 {
			return c + 3 + 3
		}
		idx := z[zIdx]
		z[zIdx]++
		n := s.stream(0, idx)
		c += 2 + 3 + 5 + 5 + cross(s.ptrOf(0), idx)
		switch n {
		case 0:
			c += 3 // BEQ taken
			silent = true
		case 0xff:
			c += 2 + 2 + 2 + 2 + 3 + 3
			z[zEnd] = 0xff
			z[zVol] = 0x0c
			z[zVol+2] = 0x0b
			z[zBusyTune] = 0
			c += 2 + 3 + 2 + 3 + 2 + 3
			silent, rest = true, true
		default:
			c += 2 + 2 + 3 + 6
			c += s.setPeriod(n)
		}
		_ = rest
	} else {
		c = 3 // LDA $15
		if z[zEnd] != 0 {
			c += 3
			silent = true
		} else {
			c += 2 + 3 // BNE, LDA hold
			if z[zHold+v] != 0 {
				return c + 3 + 3
			}
			c += 2 + 3 + 5 + 5
			idx := z[zIdx+v]
			z[zIdx+v]++
			n := s.stream(v, idx)
			c += cross(s.ptrOf(v), idx)
			if n == 0 {
				c += 3
				silent = true
			} else {
				c += 2 + 6
				c += s.setPeriod(n)
			}
		}
	}
	if silent {
		z[zPeriodLo], z[zPeriodHi] = 0, 0
		c += 2 + 3 + 3 + 3
	}
	r := byte(2 * v)
	s.emit(s.Now+uint64(at+c+9), 0, r, z[zPeriodLo])
	s.emit(s.Now+uint64(at+c+22), 0, r+1, z[zPeriodHi])
	s.emit(s.Now+uint64(at+c+41), 0, 8+byte(v), z[zVol+v])
	return c + 45 + 3
}

func (s *Seq) ptrOf(v int) uint16 {
	return uint16(s.z[zPtr+2*v]) | uint16(s.z[zPtr+2*v+1])<<8
}

// setPeriod is f21e: converts a note byte (octave, semitone) to the tone period.
func (s *Seq) setPeriod(n byte) (c int) {
	z := &s.z
	c = 3 + 2 + 2 + 4 + 3 + 4 + 3 + 4 + 8
	z[zPeriodLo] = s.rb(0xf23d + uint16(n&0x0f))
	z[zPeriodHi] = s.rb(0xf249 + uint16(n&0x0f))
	if n>>4 == 0 {
		return c + 3 + 6
	}
	c += 2 + 2
	for x := n >> 4; x > 0; x-- {
		z[zPeriodLo] = z[zPeriodLo]>>1 | z[zPeriodHi]&1<<7
		z[zPeriodHi] >>= 1
		c += 2 + 5 + 5 + 2 + 3
	}
	return c - 1 + 6
}

// durations counts down the note lengths of the three voices (phase 3).
func (s *Seq) durations() (c int) {
	z := &s.z
	c = 3 + 3 // JMP f18c, LDA $15
	if z[zEnd] != 0 {
		return c + 3 + s.restart()
	}
	c += 2
	for v := 0; v < 3; v++ {
		c += 3 // LDA counter
		if z[zDur+v] == 0 {
			c += 2 + 3 + 5 + 3 + 5 + 2 + 3
			if v == 0 {
				c += 0
			}
			z[zDur+v] = s.stream(v, z[zIdx+v])
			c += cross(s.ptrOf(v), z[zIdx+v])
			z[zIdx+v]++
			z[zHold+v] = 0xff
		} else {
			c += 3 + 5 + 3
			z[zDur+v]--
			if z[zDur+v] == 0 {
				c += 2 + 2 + 3 + 3
				z[zHold+v] = 0
			} else {
				c += 3
			}
		}
		c += 3 + 3 // JMP back, JMP to the next voice
	}
	return c
}

// restart (f1b0) starts the tune again after the end marker.
func (s *Seq) restart() (c int) {
	z := &s.z
	for i := 0; i < 0x0b; i++ {
		z[zIdx+i] = 0
	}
	c = 2 + 2 + 11*(4+2+2) + 2 + 3
	base := uint16(0xf380)
	if z[zTune] != 0 {
		base = 0xf386
		z[0x1f] = 0x0c
	}
	for i := 0; i < 6; i++ {
		z[zPtr+i] = s.rb(base + uint16(i))
	}
	return c + 6*(4+4+2+2) + 3 + 5 + 49
}
