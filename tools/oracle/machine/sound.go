package machine

import (
	"fmt"

	"burgertime/tools/oracle/m6502"
)

// Sound CPU clock is 500 kHz: one sound cycle per three main cycles.
const SoundDiv = 3

// AYWrite is one write to an AY-3-8910 register, stamped with the sound CPU cycle.
type AYWrite struct {
	Chip  int
	Reg   byte
	Val   byte
	Cycle uint64
}

// Sound is the audio board: 6502 + 1 KB RAM + 2 x AY-3-8910 + command latch.
type Sound struct {
	CPU       m6502.CPU
	RAM       [0x400]byte
	ROM       [0x1000]byte
	Latch     byte
	Pending   bool
	NMIEnable bool
	nmiIn     bool // output of the scanline divider (1 during lines with bit 3 set)
	nmiOut    bool
	ayAddr    [2]byte
	AYRegs    [2][16]byte
	OnAY      func(w AYWrite)
	Log       []AYWrite
	Record    bool
	Reads     int
}

func newSound(rs ROMSet) (*Sound, error) {
	s := &Sound{}
	b, ok := rs["ab14.12h"]
	if !ok || len(b) != 0x1000 {
		return nil, fmt.Errorf("missing/short sound ROM ab14.12h")
	}
	copy(s.ROM[:], b)
	s.CPU.Bus = s
	return s, nil
}

func (s *Sound) Read(a uint16) byte {
	switch {
	case a < 0x2000:
		return s.RAM[a&0x3ff]
	case a >= 0xa000 && a < 0xc000:
		s.Pending = false
		s.CPU.IRQ = false
		s.Reads++
		return s.Latch
	case a >= 0xe000:
		return s.ROM[a&0xfff]
	}
	return 0xff
}

func (s *Sound) Write(a uint16, v byte) {
	switch {
	case a < 0x2000:
		s.RAM[a&0x3ff] = v
	case a >= 0x2000 && a < 0x4000:
		s.ayData(0, v)
	case a >= 0x4000 && a < 0x6000:
		s.ayAddr[0] = v & 0xf
	case a >= 0x6000 && a < 0x8000:
		s.ayData(1, v)
	case a >= 0x8000 && a < 0xa000:
		s.ayAddr[1] = v & 0xf
	case a >= 0xc000 && a < 0xe000:
		s.NMIEnable = v&1 != 0
		s.updNMI()
	}
}

func (s *Sound) ayData(chip int, v byte) {
	r := s.ayAddr[chip]
	s.AYRegs[chip][r] = v
	w := AYWrite{chip, r, v, s.CPU.Cycles}
	if s.Record {
		s.Log = append(s.Log, w)
	}
	if s.OnAY != nil {
		s.OnAY(w)
	}
}

// Command latches a command from the main CPU and raises the sound CPU's IRQ.
func (s *Sound) Command(v byte) {
	s.Latch = v
	s.Pending = true
	s.CPU.IRQ = true
}

// SetDivider sets the scanline divider output (bit 3 of the scanline number).
func (s *Sound) SetDivider(hi bool) {
	s.nmiIn = hi
	s.updNMI()
}

func (s *Sound) updNMI() {
	out := s.nmiIn && s.NMIEnable
	if out && !s.nmiOut {
		s.CPU.NMI = true
	}
	s.nmiOut = out
}
