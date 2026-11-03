// Package game is the native Go rewrite of the Burger Time main-CPU program.
//
// The game keeps the arcade board's memory layout (zero page variables, object
// tables, video RAM, palette RAM) so that every routine can be proven equivalent
// to the original program by running both side by side. Routine names carry the
// address of the original code (sub_xxxx) as a cross reference.
//
// The original program is a single thread that blocks on the vertical blank
// flag in several places (title screens, "get ready" pauses, game over, ...).
// The rewrite keeps that structure: the game runs on its own goroutine and
// blocks in waitFrame(); the host calls Frame() once per video frame.
package game

import (
	"burgertime/internal/assets"
)

// Input bits (active low, as on the real board unless noted).
const (
	InRight  = 0x01
	InLeft   = 0x02
	InUp     = 0x04
	InDown   = 0x08
	InButton = 0x10 // pepper

	SysStart1 = 0x01 // active low
	SysStart2 = 0x02 // active low
	SysTilt   = 0x04 // active low
	SysCoin1  = 0x40 // active HIGH
	SysCoin2  = 0x80 // active HIGH
)

// DIP switch defaults: 3 lives, 20000 bonus, 4 enemies, end-of-level pepper, upright.
const (
	DefaultDSW1 = 0x3f
	DefaultDSW2 = 0xeb
)

// Game is the whole machine state visible to the main program.
type Game struct {
	RAM     [0x800]byte // $0000-$07ff work RAM (zero page, stack page, object tables)
	VRAM    [0x400]byte // $1000-$13ff character codes; row 0 of the map doubles as sprite RAM
	CRAM    [0x400]byte // $1400-$17ff character colour/bank
	Palette [16]byte    // $0c00-$0c0f
	Scroll  byte        // $4004 write: background layer control
	VidCtl  byte        // $4002 write: bit 0 flips the screen

	// Input ports as the program reads them.
	P1, P2, System byte
	DSW1, DSW2     byte

	// Sound commands written to $4003, in order; the host drains them.
	SndCmds []byte

	rom [0x5000]byte // $b000-$ffff program ROM (tables only; no code is executed from it)

	// carry is the processor carry flag as the original program leaves it between
	// routines; the random number generator mixes it in.
	carry bool

	// coinIRQ is the pending coin interrupt request.
	coinIRQ bool

	// frame synchronisation
	resume  chan struct{}
	yield   chan struct{}
	started bool
	// Relaxed forgives near misses of the four-way joystick (see relaxed.go).
	Relaxed bool

	// Dev hooks (nil in the shipped game).
	Legacy Legacy

	// Frames counts the video frames (vertical blank starts) seen by the program.
	Frames int
}

// Legacy lets development builds run not yet ported routines on a reference CPU.
type Legacy interface {
	// Call runs the original routine at addr with the given registers and returns the registers afterwards.
	Call(g *Game, addr uint16, r Regs) Regs
}

// Regs carries the 6502 registers across a call, for routines that return values in them.
type Regs struct {
	A, X, Y byte
	P       byte // processor status; only the flags a routine documents as results are meaningful
}

const (
	FlagC = 0x01
	FlagZ = 0x02
	FlagN = 0x80
)

// New creates a game around the program ROM data.
func New(r *assets.ROMs) *Game {
	g := &Game{}
	copy(g.rom[:], r.Main[:])
	g.P1, g.P2 = 0xff, 0xff
	g.System = 0x07 | 0x00
	g.DSW1, g.DSW2 = DefaultDSW1, DefaultDSW2
	g.resume = make(chan struct{})
	g.yield = make(chan struct{})
	return g
}

// Rom returns a byte of the program ROM (used for lookup tables and text).
func (g *Game) Rom(a uint16) byte { return g.rom[a-0xb000] }

// mirror swaps the x and y halves of a 10-bit screen offset (see the board's $1800/$1c00 windows).
func mirror(o uint16) uint16 { return (o>>5)&0x1f | (o&0x1f)<<5 }

// Rd reads a byte as the CPU would.
func (g *Game) Rd(a uint16) byte {
	switch {
	case a < 0x800:
		return g.RAM[a]
	case a >= 0xc00 && a < 0xc10:
		return g.Palette[a-0xc00]
	case a >= 0x1000 && a < 0x1400:
		return g.VRAM[a-0x1000]
	case a >= 0x1400 && a < 0x1800:
		return g.CRAM[a-0x1400]
	case a >= 0x1800 && a < 0x1c00:
		return g.VRAM[mirror(a-0x1800)]
	case a >= 0x1c00 && a < 0x2000:
		return g.CRAM[mirror(a-0x1c00)]
	case a == 0x4000:
		return g.P1
	case a == 0x4001:
		return g.P2
	case a == 0x4002:
		return g.System
	case a == 0x4003:
		return g.DSW1 &^ 0x80
	case a == 0x4004:
		return g.DSW2
	case a >= 0xb000:
		return g.rom[a-0xb000]
	}
	return 0xff
}

// Wr writes a byte as the CPU would.
func (g *Game) Wr(a uint16, v byte) {
	switch {
	case a < 0x800:
		g.RAM[a] = v
	case a >= 0xc00 && a < 0xc10:
		g.Palette[a-0xc00] = v
	case a >= 0x1000 && a < 0x1400:
		g.VRAM[a-0x1000] = v
	case a >= 0x1400 && a < 0x1800:
		g.CRAM[a-0x1400] = v
	case a >= 0x1800 && a < 0x1c00:
		g.VRAM[mirror(a-0x1800)] = v
	case a >= 0x1c00 && a < 0x2000:
		g.CRAM[mirror(a-0x1c00)] = v
	case a == 0x4002:
		g.VidCtl = v
	case a == 0x4003:
		g.SndCmds = append(g.SndCmds, v)
	case a == 0x4004:
		g.Scroll = v
	}
}

// ptr reads the 16-bit pointer stored at zero page address z.
func (g *Game) ptr(z byte) uint16 { return uint16(g.RAM[z]) | uint16(g.RAM[z+1])<<8 }

// setPtr stores a 16-bit pointer at zero page address z.
func (g *Game) setPtr(z byte, v uint16) { g.RAM[z] = byte(v); g.RAM[z+1] = byte(v >> 8) }

// Frame advances the game by one video frame: it runs from one start of vertical
// blank to the next.
func (g *Game) Frame() {
	if !g.started {
		g.started = true
		go func() {
			<-g.resume
			g.reset()
		}()
	}
	g.resume <- struct{}{}
	<-g.yield
}

// waitVblank blocks the game until the next start of vertical blank.
func (g *Game) waitVblank() {
	g.yield <- struct{}{}
	<-g.resume
}

// Coin signals a coin insertion (the board raises an interrupt request).
func (g *Game) Coin() { g.coinIRQ = true }
