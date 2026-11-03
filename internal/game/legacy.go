package game

import "fmt"

// call runs a routine that has not been ported yet on the development reference CPU.
// The shipped build contains no such calls.
func (g *Game) call(addr uint16, r Regs) Regs {
	if g.Legacy == nil {
		panic(fmt.Sprintf("game: routine %04x is not ported", addr))
	}
	r.P = r.P&^FlagC | b2f(g.carry)
	out := g.Legacy.Call(g, addr, r)
	g.carry = out.P&FlagC != 0
	return out
}

func b2f(b bool) byte {
	if b {
		return FlagC
	}
	return 0
}

// Carry reports the carry flag the program has left behind (it feeds the random number generator).
func (g *Game) Carry() byte { return b2f(g.carry) }

// SetCarry is used by the development tools when handing control between the reference CPU and ported code.
func (g *Game) SetCarry(c byte) { g.carry = c&FlagC != 0 }

// unwind is the panic value used for the original program's non-local jumps
// (it resets the stack and jumps to a fixed point of the main program).
type unwind struct{ to uint16 }

// jumpTo abandons whatever the game is doing and continues at one of the main
// program's entry points (stack reset + JMP in the original).
func (g *Game) jumpTo(addr uint16) { panic(unwind{addr}) }

// JumpTo is the exported form of jumpTo for the development tools.
func (g *Game) JumpTo(addr uint16) { g.jumpTo(addr) }
