// Package machine is the DEVELOPMENT-ONLY reference Burger Time board: main 6502,
// RAM, video RAM, palette RAM, inputs. The shipped game does not use it.
package machine

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"burgertime/tools/oracle/m6502"
)

const (
	MainClock   = 1500000 // Hz
	LineCycles  = 96      // 384 pixels @ 6 MHz / (1.5 MHz)
	Lines       = 272
	FrameCycles = LineCycles * Lines
	VblankStart = 248
	VblankEnd   = 8
)

// ROMSet holds the raw ROM images by file name.
type ROMSet map[string][]byte

// LoadROMs reads a directory or zip of ROM files.
func LoadROMs(path string) (ROMSet, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	rs := ROMSet{}
	if st.IsDir() {
		es, _ := os.ReadDir(path)
		for _, e := range es {
			b, err := os.ReadFile(filepath.Join(path, e.Name()))
			if err != nil {
				return nil, err
			}
			rs[e.Name()] = b
		}
		return rs, nil
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	for _, f := range zr.File {
		r, _ := f.Open()
		b, _ := io.ReadAll(r)
		r.Close()
		rs[filepath.Base(f.Name)] = b
	}
	return rs, nil
}

// Machine is the main board.
type Machine struct {
	CPU      m6502.CPU
	RAM      [0x800]byte
	Palette  [16]byte
	VRAM     [0x400]byte
	CRAM     [0x400]byte
	ROM      [0x10000]byte
	P1, P2   byte // active-low inputs
	System   byte
	DSW1     byte // bit 7 is overwritten by vblank
	DSW2     byte
	Scroll   [2]byte
	VidCtl   byte
	SndLatch byte
	SndWrite func(v byte)
	NoSound  bool // skip running the sound board (faster)
	Line     int
	LineCyc  int
	Frame    uint64
	Trace    func(pc uint16)
	// Access log hooks
	OnRead     func(a uint16, v byte)
	OnWrite    func(a uint16, v byte)
	WriteLog   map[uint16]int
	hadWritten bool
	Snd        *Sound
	SndCarry   int
	CoinIRQ    bool
}

func New(rs ROMSet) (*Machine, error) {
	m := &Machine{}
	for _, l := range []struct {
		n string
		a int
	}{{"ab05a1.12b", 0xb000}, {"ab04.9b", 0xc000}, {"ab06.13b", 0xd000}, {"ab05.10b", 0xe000}, {"ab07.15b", 0xf000}} {
		b, ok := rs[l.n]
		if !ok || len(b) != 0x1000 {
			return nil, fmt.Errorf("missing/short ROM %s", l.n)
		}
		copy(m.ROM[l.a:], b)
	}
	m.P1, m.P2, m.System = 0xff, 0xff, 0x07
	m.DSW1, m.DSW2 = 0x3f, 0xeb
	m.CPU.Bus = m
	m.CPU.OpFilter = m.opFilter
	m.CPU.HoldIRQ = true
	snd, err := newSound(rs)
	if err != nil {
		return nil, err
	}
	m.Snd = snd
	snd.CPU.Reset()
	m.SndWrite = snd.Command
	return m, nil
}

func (m *Machine) Vblank() bool { return m.Line >= VblankStart || m.Line < VblankEnd }

func (m *Machine) Read(a uint16) byte {
	var v byte
	switch {
	case a < 0x800:
		v = m.RAM[a]
	case a >= 0x1000 && a < 0x1400:
		v = m.VRAM[a-0x1000]
	case a >= 0x1400 && a < 0x1800:
		v = m.CRAM[a-0x1400]
	case a >= 0x1800 && a < 0x1c00:
		v = m.VRAM[mirror(a-0x1800)]
	case a >= 0x1c00 && a < 0x2000:
		v = m.CRAM[mirror(a-0x1c00)]
	case a == 0x4000:
		v = m.P1
	case a == 0x4001:
		v = m.P2
	case a == 0x4002:
		v = m.System
	case a == 0x4003:
		v = m.DSW1 &^ 0x80
		if m.Vblank() {
			v |= 0x80
		}
	case a == 0x4004:
		v = m.DSW2
	case a >= 0xb000:
		v = m.ROM[a]
	default:
		v = 0xff
	}
	if m.OnRead != nil {
		m.OnRead(a, v)
	}
	return v
}

// mirror swaps the x and y halves of a 10-bit tile offset (offset = 32*y+x -> 32*x+y).
func mirror(o uint16) uint16 { return (o>>5)&0x1f | (o&0x1f)<<5 }

// opFilter implements the DECO CPU-7 opcode scrambling: the first opcode fetched
// after any memory write, if its address has bits 0x104 both set, has its bits
// permuted (bit7<-6, 6<-5, 5<-3, 4<-4, 3<-2, 2<-7, 1<-1, 0<-0).
func (m *Machine) opFilter(pc uint16, op byte) byte {
	if ScrambleMode >= 5 && pc == 0xe306 {
		m.hadWritten = false
		return Descramble(op)
	}
	switch ScrambleMode {
	case 1: // flag cleared only by a matching fetch
		if m.hadWritten && pc&0x104 == 0x104 {
			m.hadWritten = false
			return Descramble(op)
		}
		return op
	default:
		if m.hadWritten {
			m.hadWritten = false
			if pc&0x104 == 0x104 {
				d := Descramble(op)
				if OnDescramble != nil {
					OnDescramble(pc, op, d)
				}
				return d
			}
		}
	}
	return op
}

// Descramble applies the CPU-7 opcode bit permutation.
func Descramble(v byte) byte {
	return (v>>6&1)<<7 | (v>>5&1)<<6 | (v>>3&1)<<5 | (v>>4&1)<<4 | (v>>2&1)<<3 | (v>>7&1)<<2 | (v>>1&1)<<1 | v&1
}

// ArmMask disables arming of the scramble flag for classes of writing instruction
// (experimental): bit0 abs,X/abs,Y stores; bit1 (zp),Y/(zp,X) stores; bit2 zp stores;
// bit3 abs stores; bit4 read-modify-write; bit5 stack pushes (PHA/PHP).
var ArmMask int

// OnDescramble, if set, is told about every opcode fetch that was descrambled.
var OnDescramble func(pc uint16, raw, dec byte)

func (m *Machine) armOK() bool {
	if ArmMask == 0 {
		return true
	}
	op := m.CPU.LastOp
	switch op {
	case 0x9d, 0x99:
		return ArmMask&1 == 0
	case 0x91, 0x81:
		return ArmMask&2 == 0
	case 0x85, 0x86, 0x84, 0x95, 0x96, 0x94:
		return ArmMask&4 == 0
	case 0x8d, 0x8e, 0x8c:
		return ArmMask&8 == 0
	case 0x48, 0x08:
		return ArmMask&32 == 0
	case 0x20, 0x00:
		return true
	default:
		return ArmMask&16 == 0
	}
}

// ScrambleMode selects experimental flag semantics (0 = MAME behaviour).
var ScrambleMode = 0

func (m *Machine) Write(a uint16, v byte) {
	if !(ScrambleMode == 2 && a >= 0x100 && a < 0x200) && !(ScrambleMode == 3 && m.CPU.Quiet) && m.armOK() {
		m.hadWritten = true
	}
	if m.OnWrite != nil {
		m.OnWrite(a, v)
	}
	switch {
	case a < 0x800:
		m.RAM[a] = v
	case a >= 0xc00 && a < 0xc10:
		m.Palette[a-0xc00] = v
	case a >= 0x1000 && a < 0x1400:
		m.VRAM[a-0x1000] = v
	case a >= 0x1400 && a < 0x1800:
		m.CRAM[a-0x1400] = v
	case a >= 0x1800 && a < 0x1c00:
		m.VRAM[mirror(a-0x1800)] = v
	case a >= 0x1c00 && a < 0x2000:
		m.CRAM[mirror(a-0x1c00)] = v
	case a == 0x4000:
	case a == 0x4002:
		m.VidCtl = v
	case a == 0x4003:
		m.SndLatch = v
		if m.SndWrite != nil {
			m.SndWrite(v)
		}
	case a == 0x4004:
		m.Scroll[0] = v
	default:
		if m.WriteLog != nil {
			m.WriteLog[a]++
		}
	}
}

// Run executes n CPU cycles of the main board, tracking the raster line.
func (m *Machine) RunCycles(n int) {
	end := m.CPU.Cycles + uint64(n)
	for m.CPU.Cycles < end {
		if m.Trace != nil {
			m.Trace(m.CPU.PC)
		}
		c := m.CPU.Step()
		m.LineCyc += c
		if !m.NoSound {
			m.runSound(c)
		}
		for m.LineCyc >= LineCycles {
			m.LineCyc -= LineCycles
			m.Line++
			if m.Line >= Lines {
				m.Line = 0
				m.Frame++
			}
			m.Snd.SetDivider(m.Line&8 != 0)
		}
	}
}

func (m *Machine) runSound(mainCycles int) {
	m.SndCarry += mainCycles
	for m.SndCarry >= SoundDiv {
		// run the sound CPU until it has consumed the cycles owed
		need := m.SndCarry / SoundDiv
		used := 0
		for used < need {
			used += m.Snd.CPU.Step()
		}
		m.SndCarry -= used * SoundDiv
		if m.SndCarry < 0 {
			// overshoot: carry a negative balance
			break
		}
	}
}
