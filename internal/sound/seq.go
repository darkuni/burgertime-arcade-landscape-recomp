// Package sound is the native rewrite of the Burger Time sound board: the
// program that ran on the board's own 6502 (music sequencer, sound effects and
// command handling) plus a synthesiser for the board's two AY-3-8910 chips.
//
// The sequencer works in "sound cycles" (500 kHz): the music engine runs on the
// board's 976 Hz timer interrupt, a sound effect is a timed script that writes
// to the second chip, and commands from the main board arrive at any time.
package sound

// ClockHz is the sound CPU clock; the AY chips run at three times that.
const (
	ClockHz   = 500000
	AYClockHz = 1500000
	// nmiPeriod: the timer interrupt comes every 16 video lines of 32 sound cycles.
	nmiPeriod = 512
)

// AYWrite is one write to an AY register.
type AYWrite struct {
	Cycle uint64
	Chip  int
	Reg   byte
	Val   byte
}

type effEvent struct {
	at   int // effect-local cycle
	chip int
	reg  byte
	val  byte
	prio bool // marks the point where the effect releases its priority
}

// Seq is the sound program.
type Seq struct {
	rom [0x1000]byte
	z   [0x100]byte // the program's zero page

	Now     uint64 // sound cycles since start
	nextNMI uint64
	busy    int // cycles the main program loses to interrupt handlers

	// effect script in progress
	eff    []effEvent
	effPos int
	effEnd int
	ec     int // effect-local clock

	OnAY func(AYWrite)
	AY   [2][16]byte
}

// New creates the sequencer in the state the original program reaches after start-up.
func New(rom []byte) *Seq {
	s := &Seq{}
	copy(s.rom[:], rom)
	s.nextNMI = nmiPeriod
	s.reinit()
	return s
}

// SetNMIPhase sets the cycles until the next timer interrupt (testing).
func (s *Seq) SetNMIPhase(c uint64) { s.nextNMI = s.Now + c }

func (s *Seq) rb(a uint16) byte { return s.rom[a&0xfff] }

func (s *Seq) emit(at uint64, chip int, reg, val byte) {
	reg &= 15
	s.AY[chip][reg] = val
	if s.OnAY != nil {
		s.OnAY(AYWrite{at, chip, reg, val})
	}
}

// reinit is the program's start (f009): clear the work RAM, silence both chips.
func (s *Seq) reinit() {
	for i := 0; i < 0xf0; i++ {
		s.z[i] = 0
	}
	s.eff, s.effPos, s.effEnd, s.ec = nil, 0, 0, 0
	// f02a: both chips get the 16 start values
	for r := 0; r < 16; r++ {
		v := s.rb(0xf064 + uint16(r))
		s.emit(s.Now, 0, byte(r), v)
		s.emit(s.Now, 1, byte(r), v)
	}
	s.z[0x1e] = 8
	// f023: the second chip's 11 registers once more
	for r := 0; r < 11; r++ {
		s.emit(s.Now, 1, byte(r), s.rb(0xf064+uint16(r)))
	}
}

// Run advances the sound board by n cycles.
func (s *Seq) Run(n int) {
	target := s.Now + uint64(n)
	for s.Now < target {
		t := target
		if s.nextNMI < t {
			t = s.nextNMI
		}
		dt := int(t - s.Now)
		// the program proper runs unless an interrupt handler is using the time
		use := dt
		if use > s.busy {
			use = s.busy
		}
		s.busy -= use
		if dt > use {
			s.runEffect(s.Now+uint64(use), dt-use)
		}
		s.Now = t
		if s.Now == s.nextNMI {
			s.nextNMI += nmiPeriod
			s.nmi()
		}
	}
}

// runEffect lets the running sound effect make progress for c cycles starting at wall time w.
func (s *Seq) runEffect(w uint64, c int) {
	if s.effPos >= len(s.eff) {
		return
	}
	first := s.ec
	s.ec += c
	for s.effPos < len(s.eff) && s.eff[s.effPos].at < s.ec {
		e := s.eff[s.effPos]
		s.effPos++
		if e.prio {
			s.z[0x3e] = 0
			continue
		}
		s.emit(w+uint64(e.at-first), e.chip, e.reg, e.val)
	}
}
