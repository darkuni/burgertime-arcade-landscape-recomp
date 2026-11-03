package game

// Helpers that model the processor's carry flag. The program's random number
// generator mixes the carry left over from earlier arithmetic into the seed, so
// the routines that run before it must leave exactly the carry the original does.

// cmp is CMP: sets carry (a >= b) and reports whether the operands are equal.
func (g *Game) cmp(a, b byte) bool {
	g.carry = a >= b
	return a == b
}

// adc is ADC with the current carry.
func (g *Game) adc(a, b byte) byte {
	s := int(a) + int(b)
	if g.carry {
		s++
	}
	g.carry = s > 0xff
	return byte(s)
}

// addc is CLC; ADC.
func (g *Game) addc(a, b byte) byte {
	g.carry = false
	return g.adc(a, b)
}

// sbc is SBC with the current carry (carry set = no borrow).
func (g *Game) sbc(a, b byte) byte {
	d := int(a) - int(b)
	if !g.carry {
		d--
	}
	g.carry = d >= 0
	return byte(d)
}

// subc is SEC; SBC.
func (g *Game) subc(a, b byte) byte {
	g.carry = true
	return g.sbc(a, b)
}

func (g *Game) lsr(a byte) byte {
	g.carry = a&1 != 0
	return a >> 1
}

func (g *Game) lsr4(a byte) byte {
	g.carry = a&8 != 0
	return a >> 4
}

func (g *Game) asl(a byte) byte {
	g.carry = a&0x80 != 0
	return a << 1
}

// ret carries the registers a routine leaves behind for callers that look at them.
type ret struct {
	a, x, y byte
	z       bool
}

// flags converts the register result into 6502 status bits (Z and N from A, as after a load).
func (r ret) flags() byte {
	var p byte
	if r.z {
		p |= FlagZ
	}
	if r.a&0x80 != 0 {
		p |= FlagN
	}
	return p
}
