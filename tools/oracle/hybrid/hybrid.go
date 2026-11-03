// Package hybrid is DEVELOPMENT-ONLY: it lets a partly ported game run the
// routines that have not been rewritten yet on the reference 6502, sharing the
// game's memory, so every porting step can be verified frame by frame.
package hybrid

import (
	"fmt"

	"burgertime/internal/game"
	"burgertime/tools/oracle/machine"
	"burgertime/tools/oracle/m6502"
)

var sentinel = uint16(0xfff9) // address the reference CPU "returns" to when a call ends

// Legacy implements game.Legacy.
type Legacy struct {
	ports   map[uint16]func(g *game.Game, r *game.Regs)
	unwind  map[uint16]bool
	sp      byte // stack pointer shared by nested calls
	Trace   func(pc uint16, a, x, y byte)
	Calls   map[uint16]int
	hadWrit bool
}

func New() *Legacy {
	l := &Legacy{ports: game.PortTable(), unwind: map[uint16]bool{0xc02f: true, 0xc06a: true, 0xc077: true, 0xc00d: true}, sp: 0xff, Calls: map[uint16]int{}}
	return l
}

type bus struct {
	g *game.Game
	l *Legacy
}

func (b *bus) Read(a uint16) byte { return b.g.Rd(a) }
func (b *bus) Write(a uint16, v byte) {
	b.l.hadWrit = true
	b.g.Wr(a, v)
}

// Call runs the routine at addr on the reference CPU and returns when it returns.
func (l *Legacy) Call(g *game.Game, addr uint16, r game.Regs) game.Regs {
	l.Calls[addr]++
	cpu := &m6502.CPU{Bus: &bus{g, l}}
	cpu.A, cpu.X, cpu.Y = r.A, r.X, r.Y
	cpu.P = r.P | m6502.FU | m6502.FI
	cpu.SP = l.sp
	saved := l.sp
	// push the return address like JSR does
	l.hadWrit = false
	cpu.Bus.Write(0x100|uint16(cpu.SP), byte((sentinel-1)>>8))
	cpu.SP--
	cpu.Bus.Write(0x100|uint16(cpu.SP), byte(sentinel-1))
	cpu.SP--
	l.hadWrit = true // the JSR that got us here was a write
	cpu.PC = addr
	cpu.OpFilter = func(pc uint16, op byte) byte {
		if l.hadWrit {
			l.hadWrit = false
			if pc&0x104 == 0x104 {
				return machine.Descramble(op)
			}
		}
		return op
	}
	first := true
	for steps := 0; ; steps++ {
		if cpu.PC == sentinel {
			break
		}
		if l.Trace != nil {
			l.Trace(cpu.PC, cpu.A, cpu.X, cpu.Y)
		}
		if !first {
			if l.unwind[cpu.PC] {
				l.sp = saved
				g.JumpTo(cpu.PC)
			}
			if p, ok := l.ports[cpu.PC]; ok {
				rr := game.Regs{A: cpu.A, X: cpu.X, Y: cpu.Y, P: cpu.P}
				l.sp = cpu.SP
				g.SetCarry(rr.P)
				func() {
					defer func() {
						if r := recover(); r != nil {
							l.sp = 0xff
							panic(r)
						}
					}()
					p(g, &rr)
				}()
				rr.P = rr.P&^m6502.FC | g.Carry()
				l.sp = saved
				cpu.A, cpu.X, cpu.Y = rr.A, rr.X, rr.Y
				cpu.P = cpu.P&^(m6502.FC|m6502.FZ|m6502.FN) | rr.P&(m6502.FC|m6502.FZ|m6502.FN)
				// RTS
				cpu.SP++
				lo := uint16(g.Rd(0x100 | uint16(cpu.SP)))
				cpu.SP++
				hi := uint16(g.Rd(0x100 | uint16(cpu.SP)))
				cpu.PC = (hi<<8 | lo) + 1
				l.hadWrit = false
				continue
			}
		}
		first = false
		cpu.Step()
		if cpu.Illegal != 0 {
			panic(fmt.Sprintf("hybrid: illegal opcode at %04x", cpu.LastPC))
		}
		if steps > 50_000_000 {
			panic(fmt.Sprintf("hybrid: runaway in routine %04x pc=%04x", addr, cpu.PC))
		}
	}
	l.sp = saved
	return game.Regs{A: cpu.A, X: cpu.X, Y: cpu.Y, P: cpu.P}
}
