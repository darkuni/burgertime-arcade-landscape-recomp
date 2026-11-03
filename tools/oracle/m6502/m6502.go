// Package m6502 is a cycle-counting NMOS 6502 interpreter. It is part of the
// DEVELOPMENT-ONLY reference machine used to verify the Go rewrite; the shipped
// game does not use it.
package m6502

// Bus is the CPU's view of memory.
type Bus interface {
	Read(addr uint16) byte
	Write(addr uint16, v byte)
}

const (
	FC = 0x01
	FZ = 0x02
	FI = 0x04
	FD = 0x08
	FB = 0x10
	FU = 0x20
	FV = 0x40
	FN = 0x80
)

type CPU struct {
	A, X, Y, SP, P byte
	PC             uint16
	Bus            Bus
	Cycles         uint64
	IRQ            bool // level-sensitive
	HoldIRQ        bool // clear the IRQ request when it is taken (HOLD_LINE behaviour)
	NMI            bool // set to request; cleared when taken
	Halted         bool // illegal/KIL opcode
	Illegal        int
	LastOp         byte
	Quiet          bool // pushes made now do not arm the CPU-7 scramble flag (JSR/BRK/interrupt)
	LastPC         uint16
	// Coverage hooks (optional)
	Fetch func(pc uint16)
	// OpFilter, if set, can alter the opcode byte fetched at pc (used for the
	// DECO CPU-7 opcode scrambling).
	OpFilter func(pc uint16, op byte) byte
}

func (c *CPU) Reset() {
	c.SP = 0xfd
	c.P = FI | FU
	c.PC = uint16(c.Bus.Read(0xfffc)) | uint16(c.Bus.Read(0xfffd))<<8
	c.Cycles += 7
}

func (c *CPU) rd16(a uint16) uint16 {
	return uint16(c.Bus.Read(a)) | uint16(c.Bus.Read(a+1))<<8
}
func (c *CPU) push(v byte) { c.Bus.Write(0x100|uint16(c.SP), v); c.SP-- }
func (c *CPU) pop() byte   { c.SP++; return c.Bus.Read(0x100 | uint16(c.SP)) }
func (c *CPU) nz(v byte) byte {
	c.P &^= FN | FZ
	if v == 0 {
		c.P |= FZ
	}
	c.P |= v & FN
	return v
}
func (c *CPU) fetch() byte { v := c.Bus.Read(c.PC); c.PC++; return v }
func (c *CPU) fetch16() uint16 {
	lo := uint16(c.fetch())
	return lo | uint16(c.fetch())<<8
}

func (c *CPU) interrupt(vec uint16, brk bool) {
	c.Quiet = true
	defer func() { c.Quiet = false }()
	c.push(byte(c.PC >> 8))
	c.push(byte(c.PC))
	p := c.P | FU
	if brk {
		p |= FB
	} else {
		p &^= FB
	}
	c.push(p)
	c.P |= FI
	c.PC = c.rd16(vec)
	c.Cycles += 7
}

// Step executes one instruction (or takes a pending interrupt) and returns the cycles used.
func (c *CPU) Step() int {
	start := c.Cycles
	if c.NMI {
		c.NMI = false
		c.interrupt(0xfffa, false)
		return int(c.Cycles - start)
	}
	if c.IRQ && c.P&FI == 0 {
		if c.HoldIRQ {
			c.IRQ = false
		}
		c.interrupt(0xfffe, false)
		return int(c.Cycles - start)
	}
	if c.Fetch != nil {
		c.Fetch(c.PC)
	}
	pc0 := c.PC
	op := c.fetch()
	if c.OpFilter != nil {
		op = c.OpFilter(pc0, op)
	}
	c.LastOp, c.LastPC = op, pc0
	cy := int(cycTab[op])
	var ea uint16
	var crossed bool
	switch modeTab[op] {
	case mImp, mAcc:
		c.Bus.Read(c.PC) // dummy
	case mImm:
		ea = c.PC
		c.PC++
	case mZp:
		ea = uint16(c.fetch())
	case mZpx:
		ea = uint16(c.fetch() + c.X)
	case mZpy:
		ea = uint16(c.fetch() + c.Y)
	case mAbs:
		ea = c.fetch16()
	case mAbx:
		b := c.fetch16()
		ea = b + uint16(c.X)
		crossed = (b^ea)&0xff00 != 0
	case mAby:
		b := c.fetch16()
		ea = b + uint16(c.Y)
		crossed = (b^ea)&0xff00 != 0
	case mIzx:
		z := c.fetch() + c.X
		ea = uint16(c.Bus.Read(uint16(z))) | uint16(c.Bus.Read(uint16(z+1)))<<8
	case mIzy:
		z := c.fetch()
		b := uint16(c.Bus.Read(uint16(z))) | uint16(c.Bus.Read(uint16(z+1)))<<8
		ea = b + uint16(c.Y)
		crossed = (b^ea)&0xff00 != 0
	case mInd:
		p := c.fetch16()
		lo := c.Bus.Read(p)
		hi := c.Bus.Read((p & 0xff00) | uint16(byte(p)+1)) // NMOS page-wrap bug
		ea = uint16(lo) | uint16(hi)<<8
	case mRel:
		ea = c.PC
		c.PC++
	}
	if crossed && pcPenalty[op] {
		cy++
	}
	rdv := func() byte { return c.Bus.Read(ea) }
	wr := func(v byte) { c.Bus.Write(ea, v) }
	// read-modify-write helper
	rmw := func(f func(byte) byte) {
		if modeTab[op] == mAcc {
			c.A = f(c.A)
			return
		}
		v := rdv()
		c.Bus.Write(ea, v) // dummy write of old value
		wr(f(v))
	}
	branch := func(cond bool) {
		off := int8(c.Bus.Read(ea))
		if cond {
			cy++
			t := c.PC + uint16(int16(off))
			if (t^c.PC)&0xff00 != 0 {
				cy++
			}
			c.PC = t
		}
	}
	switch op {
	// ---- loads/stores
	case 0xa9, 0xa5, 0xb5, 0xad, 0xbd, 0xb9, 0xa1, 0xb1:
		c.A = c.nz(rdv())
	case 0xa2, 0xa6, 0xb6, 0xae, 0xbe:
		c.X = c.nz(rdv())
	case 0xa0, 0xa4, 0xb4, 0xac, 0xbc:
		c.Y = c.nz(rdv())
	case 0x85, 0x95, 0x8d, 0x9d, 0x99, 0x81, 0x91:
		if modeTab[op] == mAbx || modeTab[op] == mAby || modeTab[op] == mIzy {
			c.Bus.Read(ea) // dummy read
		}
		wr(c.A)
	case 0x86, 0x96, 0x8e:
		wr(c.X)
	case 0x84, 0x94, 0x8c:
		wr(c.Y)
	// ---- transfers
	case 0xaa:
		c.X = c.nz(c.A)
	case 0xa8:
		c.Y = c.nz(c.A)
	case 0x8a:
		c.A = c.nz(c.X)
	case 0x98:
		c.A = c.nz(c.Y)
	case 0xba:
		c.X = c.nz(c.SP)
	case 0x9a:
		c.SP = c.X
	// ---- stack
	case 0x48:
		c.push(c.A)
	case 0x08:
		c.push(c.P | FB | FU)
	case 0x68:
		c.A = c.nz(c.pop())
	case 0x28:
		c.P = c.pop()&^FB | FU
	// ---- logic
	case 0x29, 0x25, 0x35, 0x2d, 0x3d, 0x39, 0x21, 0x31:
		c.A = c.nz(c.A & rdv())
	case 0x09, 0x05, 0x15, 0x0d, 0x1d, 0x19, 0x01, 0x11:
		c.A = c.nz(c.A | rdv())
	case 0x49, 0x45, 0x55, 0x4d, 0x5d, 0x59, 0x41, 0x51:
		c.A = c.nz(c.A ^ rdv())
	case 0x24, 0x2c:
		v := rdv()
		c.P &^= FN | FV | FZ
		c.P |= v & (FN | FV)
		if v&c.A == 0 {
			c.P |= FZ
		}
	// ---- arithmetic
	case 0x69, 0x65, 0x75, 0x6d, 0x7d, 0x79, 0x61, 0x71:
		c.adc(rdv())
	case 0xe9, 0xe5, 0xf5, 0xed, 0xfd, 0xf9, 0xe1, 0xf1:
		c.sbc(rdv())
	case 0xc9, 0xc5, 0xd5, 0xcd, 0xdd, 0xd9, 0xc1, 0xd1:
		c.cmp(c.A, rdv())
	case 0xe0, 0xe4, 0xec:
		c.cmp(c.X, rdv())
	case 0xc0, 0xc4, 0xcc:
		c.cmp(c.Y, rdv())
	case 0xe6, 0xf6, 0xee, 0xfe:
		rmw(func(v byte) byte { return c.nz(v + 1) })
	case 0xc6, 0xd6, 0xce, 0xde:
		rmw(func(v byte) byte { return c.nz(v - 1) })
	case 0xe8:
		c.X = c.nz(c.X + 1)
	case 0xca:
		c.X = c.nz(c.X - 1)
	case 0xc8:
		c.Y = c.nz(c.Y + 1)
	case 0x88:
		c.Y = c.nz(c.Y - 1)
	// ---- shifts
	case 0x0a, 0x06, 0x16, 0x0e, 0x1e:
		rmw(func(v byte) byte { c.P = c.P&^FC | v>>7; return c.nz(v << 1) })
	case 0x4a, 0x46, 0x56, 0x4e, 0x5e:
		rmw(func(v byte) byte { c.P = c.P&^FC | v&1; return c.nz(v >> 1) })
	case 0x2a, 0x26, 0x36, 0x2e, 0x3e:
		rmw(func(v byte) byte {
			cin := c.P & FC
			c.P = c.P&^FC | v>>7
			return c.nz(v<<1 | cin)
		})
	case 0x6a, 0x66, 0x76, 0x6e, 0x7e:
		rmw(func(v byte) byte {
			cin := c.P & FC
			c.P = c.P&^FC | v&1
			return c.nz(v>>1 | cin<<7)
		})
	// ---- flags
	case 0x18:
		c.P &^= FC
	case 0x38:
		c.P |= FC
	case 0x58:
		c.P &^= FI
	case 0x78:
		c.P |= FI
	case 0xb8:
		c.P &^= FV
	case 0xd8:
		c.P &^= FD
	case 0xf8:
		c.P |= FD
	// ---- branches
	case 0x10:
		branch(c.P&FN == 0)
	case 0x30:
		branch(c.P&FN != 0)
	case 0x50:
		branch(c.P&FV == 0)
	case 0x70:
		branch(c.P&FV != 0)
	case 0x90:
		branch(c.P&FC == 0)
	case 0xb0:
		branch(c.P&FC != 0)
	case 0xd0:
		branch(c.P&FZ == 0)
	case 0xf0:
		branch(c.P&FZ != 0)
	// ---- jumps
	case 0x4c, 0x6c:
		c.PC = ea
	case 0x20:
		// JSR pushes address of last byte of the instruction
		c.PC--
		c.Quiet = true
		c.push(byte(c.PC >> 8))
		c.push(byte(c.PC))
		c.Quiet = false
		c.PC = ea
	case 0x60:
		lo := uint16(c.pop())
		c.PC = (lo | uint16(c.pop())<<8) + 1
	case 0x40:
		c.P = c.pop()&^FB | FU
		lo := uint16(c.pop())
		c.PC = lo | uint16(c.pop())<<8
	case 0x00:
		c.PC++
		c.interrupt(0xfffe, true)
		return int(c.Cycles - start)
	case 0xea:
	default:
		if undocNOP[op] {
			if modeTab[op] != mImp {
				c.Bus.Read(ea)
			}
			break
		}
		c.Illegal++
		// treat as NOP of the proper length for robustness; flag it
		c.Halted = illegalHalts[op]
	}
	c.Cycles += uint64(cy)
	return int(c.Cycles - start)
}

func (c *CPU) cmp(r, v byte) {
	c.P &^= FC | FN | FZ
	if r >= v {
		c.P |= FC
	}
	d := r - v
	if d == 0 {
		c.P |= FZ
	}
	c.P |= d & FN
}

func (c *CPU) adc(v byte) {
	a := c.A
	cin := c.P & FC
	if c.P&FD != 0 {
		lo := int(a&0xf) + int(v&0xf) + int(cin)
		hi := int(a>>4) + int(v>>4)
		// Z flag from binary result on NMOS
		bin := int(a) + int(v) + int(cin)
		c.P &^= FZ | FN | FV | FC
		if byte(bin) == 0 {
			c.P |= FZ
		}
		if lo > 9 {
			lo += 6
		}
		if lo > 0xf {
			hi++
		}
		c.P |= byte(hi<<4) & FN
		if (^(a ^ v))&(a^byte(hi<<4))&0x80 != 0 {
			c.P |= FV
		}
		if hi > 9 {
			hi += 6
		}
		if hi > 0xf {
			c.P |= FC
		}
		c.A = byte(hi<<4) | byte(lo&0xf)
		return
	}
	s := int(a) + int(v) + int(cin)
	r := byte(s)
	c.P &^= FC | FV
	if s > 0xff {
		c.P |= FC
	}
	if (^(a ^ v))&(a^r)&0x80 != 0 {
		c.P |= FV
	}
	c.A = c.nz(r)
}

func (c *CPU) sbc(v byte) {
	a := c.A
	cin := int(c.P & FC)
	bin := int(a) - int(v) - (1 - cin)
	if c.P&FD != 0 {
		lo := int(a&0xf) - int(v&0xf) - (1 - cin)
		hi := int(a>>4) - int(v>>4)
		if lo < 0 {
			lo -= 6
			hi--
		}
		if hi < 0 {
			hi -= 6
		}
		r := byte(bin)
		c.P &^= FC | FV
		if bin >= 0 {
			c.P |= FC
		}
		if (a^v)&(a^r)&0x80 != 0 {
			c.P |= FV
		}
		c.nz(r)
		c.A = byte(hi<<4) | byte(lo&0xf)
		return
	}
	r := byte(bin)
	c.P &^= FC | FV
	if bin >= 0 {
		c.P |= FC
	}
	if (a^v)&(a^r)&0x80 != 0 {
		c.P |= FV
	}
	c.A = c.nz(r)
}
