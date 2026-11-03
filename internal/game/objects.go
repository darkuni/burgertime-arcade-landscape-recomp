package game

// Object animation and movement helpers (sub_d36c .. sub_d78e).
//
// Up to eight objects (the player, the enemies, ...) live in parallel tables:
//   $68+x state/frame-list number, $91+x animation speed mask, $99+x countdown,
//   $a1+x countdown reload, $a9+x direction (high nibble) and frame (low bits).
// Their sprite data lives in the sprite RAM window at $1800+4*x:
//   +0 attributes, +1 tile code, +2 and +3 the two coordinates.

// sub_d36c advances the animation (and movement) of object x.
func (g *Game) sub_d36c(x byte) {
	R := &g.RAM
	if R[0x68+x]&0x80 != 0 {
		return
	}
	if R[0x13]&R[0x91+x] == 0 {
		return
	}
	R[0x99+x]--
	if R[0x99+x] != 0 {
		return
	}
	R[0x99+x] = R[0xa1+x]
	a9 := R[0xa9+x]
	R[3] = a9 & 0x0f
	a := g.lsr(g.lsr(g.lsr(a9)))
	if a != 0 {
		R[3] = g.addc(a, R[3])
	} else {
		g.cmp(x, 7)
		if x == 7 {
			y := byte(0)
			g.cmp(R[0xba], 6)
			if R[0xba] == 6 {
				y = 1
			}
			R[3] = y
		}
	}
	// d3af
	g.asl(g.asl(x))
	R[9] = x << 2
	yy := g.asl(R[0x68+x] & 7) // index into the frame list pointer table
	g.setPtr(5, uint16(g.Rom(0xd44e+uint16(yy)))|uint16(g.Rom(0xd44f+uint16(yy)))<<8)
	f := g.Rd(g.ptr(5) + uint16(R[3]))
	g.cmp(f, 0xff)
	y := R[9]
	if f != 0xff {
		g.Wr(0x1801+uint16(y), f)
		a9 = R[0xa9+x]
		g.carry = a9&8 != 0
		g.Wr(0x1800+uint16(y), (a9>>4)&2|1)
		if a9&0xf0 == 0 {
			return
		}
		R[0xa9+x]++
		R[0xa9+x] &= 0xf3
		a = R[0xa9+x]
		R[8] = x
		d := a >> 4
		g.carry = false
		v := g.addc(g.Rom(0xd4ba+uint16(d)), g.Rd(0x1802+uint16(y)))
		g.Wr(0x1802+uint16(y), v)
		v = g.addc(g.Rom(0xd4bb+uint16(d)), g.Rd(0x1803+uint16(y)))
		if g.cmp(v, 0x1d); !g.carry {
			v = 0x1d
		}
		g.Wr(0x1803+uint16(y), v)
		return
	}
	// end of the frame list: keep creeping along the second coordinate
	if R[0xa9+x]&0x80 == 0 {
		R[0x0b] = g.Rd(0x1803+uint16(y)) + 1
	} else {
		R[0x0b] = g.Rd(0x1803+uint16(y)) - 1
	}
	g.Wr(0x1803+uint16(y), R[0x0b])
	if R[0xa9+x]&0xf0 != 0 {
		R[0xa9+x]++
		R[0xa9+x] &= 0xf3
	}
}

// sub_d4c6 moves object x one step according to its direction and checks the
// maze: it dispatches on the direction nibble of $a9,x.
func (g *Game) sub_d4c6(x byte) ret {
	R := &g.RAM
	d := g.lsr4(R[0xa9+x])
	p := uint16(g.Rom(0xd4eb+uint16(d))) | uint16(g.Rom(0xd4ec+uint16(d)))<<8
	g.setPtr(5, p)
	g.asl(g.asl(x))
	y := x << 2
	R[3] = g.Rd(0x1802 + uint16(y))
	R[4] = g.Rd(0x1803 + uint16(y))
	R[0x67]++
	switch p {
	case 0xd4f8:
		R[0x67] = 0
		return ret{0, x, y, true}
	case 0xd500:
		return g.sub_d500(x, y)
	case 0xd531:
		return g.sub_d531(x, y)
	case 0xd562:
		return g.sub_d562(x, y)
	case 0xd5b0:
		return g.sub_d5b0(x, y)
	}
	panic("sub_d4c6: bad direction")
}

func (g *Game) sub_d500(x, y byte) ret {
	R := &g.RAM
	if !g.cmpGE(R[3], 0x16) {
		g.Wr(0x1802+uint16(y), 0x17)
		return ret{0x17, x, y, false}
	}
	a, z := g.sub_d603()
	if !z {
		return ret{a, x, y, false}
	}
	a, x = g.sub_d68b()
	if a != 0 {
		return ret{a, x, y, false}
	}
	g.Wr(0x1803+uint16(y), g.subc(R[4], 1)&0xf0|0x0d)
	R[0x67] = 0
	return ret{0, x, y, true}
}

// cmpGE is CMP with a branch on carry set (a >= b).
func (g *Game) cmpGE(a, b byte) bool { g.cmp(a, b); return g.carry }

func (g *Game) sub_d531(x, y byte) ret {
	R := &g.RAM
	if g.cmpGE(R[3], 0xd8) {
		g.Wr(0x1802+uint16(y), 0xd8)
		return ret{0xd8, x, y, false}
	}
	a, z := g.sub_d603()
	if !z {
		return ret{a, x, y, false}
	}
	a, x = g.sub_d6ae()
	if a != 0 {
		return ret{a, x, y, false}
	}
	g.Wr(0x1803+uint16(y), g.subc(R[4], 1)&0xf0|0x0d)
	R[0x67] = 0
	return ret{0, x, y, true}
}

func (g *Game) sub_d562(x, y byte) ret {
	R := &g.RAM
	if !g.cmpGE(R[4], 0x1d) {
		g.Wr(0x1803+uint16(y), 0x1d)
		return ret{0x1d, x, y, false}
	}
	a, z := g.sub_d626(x)
	if !z {
		return ret{a, x, y, false}
	}
	a, x = g.sub_d721()
	if a != 0 {
		return ret{a, x, y, false}
	}
	return g.finishStep(x, y)
}

func (g *Game) sub_d5b0(x, y byte) ret {
	R := &g.RAM
	if g.cmpGE(R[4], 0xdd) {
		g.Wr(0x1803+uint16(y), 0xdd)
		return ret{0xdd, x, y, false}
	}
	a, z := g.sub_d626(x)
	if !z {
		return ret{a, x, y, false}
	}
	a, x = g.sub_d744()
	if a != 0 {
		return ret{a, x, y, false}
	}
	return g.finishStep(x, y)
}

// finishStep snaps the first coordinate to the grid (d58a..d5a3 / d5da..d5f3).
func (g *Game) finishStep(x, y byte) ret {
	R := &g.RAM
	pushedX := x
	snap := R[8] & 0xf8
	xx := byte(0)
	if g.Rd(0x4003)&0x40 != 0 {
		xx = R[0x1f]
	}
	g.Wr(0x1802+uint16(y), g.addc(snap, g.Rom(0xd5fd+uint16(xx))))
	R[0x67] = 0
	return ret{0, pushedX, y, true}
}

// sub_d603: Z set when the second coordinate is on a grid column where turning is allowed.
func (g *Game) sub_d603() (a byte, z bool) {
	a = g.RAM[4] & 0x0f
	if a == 0 {
		return a, true
	}
	g.carry = a >= 0x0a
	return a, a >= 0x0a
}

// chain compares v against each value in turn, stopping at the first match, and
// leaves the carry flag of the last comparison.
func (g *Game) chain(v byte, vals ...byte) bool {
	for _, c := range vals {
		if g.cmp(v, c) {
			return true
		}
	}
	return false
}

// sub_d626: Z set when object x may continue along the current row / ladder line.
func (g *Game) sub_d626(x byte) (a byte, z bool) {
	R := &g.RAM
	R[8] = R[3]
	if g.Rd(0x4003)&0x40 != 0 && R[0x1f] != 0 {
		R[8] -= 2
	}
	R[8] += 3
	lo := R[8] & 0x0f
	switch {
	case lo < 6:
		g.carry = false
		hi := R[8] & 0xf0
		return hi, g.chain(hi, 0x30, 0x60, 0x90, 0xc0)
	case lo < 8:
		g.carry = false
		return lo, false
	case lo < 0x0e:
		g.carry = false
		hi := R[8] & 0xf0
		return hi, g.chain(hi, 0x10, 0x40, 0x70, 0xa0, 0xd0)
	}
	// The original falls into the middle of an instruction here and executes
	// SBC $ea,X: the result depends on the work RAM byte at $ea+x.
	g.carry = true
	r := g.sbc(lo, R[byte(0xea+x)])
	return r, r == 0
}

// mapCell returns the maze cell (high nibble of the byte in the level map) for a table index.
func (g *Game) mapCell(idx byte) (cell byte, x byte) {
	x = g.Rom(0xd791 + uint16(idx))
	return g.RAM[0x400+uint16(x)] >> 4, x
}

func (g *Game) sub_d68b() (a, x byte) {
	R := &g.RAM
	v := g.subc(R[3], 8)
	R[0x12] = g.lsr4(v)
	R[0x12] = g.subc(0x0f, R[0x12])
	v = g.addc(R[4], 7) & 0xf0
	return g.sub_d6cf(g.addc(v, R[0x12]))
}

func (g *Game) sub_d6ae() (a, x byte) {
	R := &g.RAM
	v := g.addc(R[3], 0x19)
	R[0x12] = g.lsr4(v)
	R[0x12] = g.subc(0x0f, R[0x12])
	v = g.addc(R[4], 7) & 0xf0
	return g.sub_d6cf(g.addc(v, R[0x12]))
}

func (g *Game) sub_d6cf(idx byte) (a, x byte) {
	R := &g.RAM
	cell, x := g.mapCell(idx)
	g.carry = false // set by the following chain
	if g.chain(cell, 0, 5, 6, 7) {
		return 0xff, x
	}
	v := g.addc(R[3], 5)
	R[0x12] = g.lsr4(v)
	R[0x12] = g.subc(0x0f, R[0x12])
	v = g.addc(R[4], 0x10) & 0xf0
	cell, x = g.mapCell(g.addc(v, R[0x12]))
	if g.cmp(cell, 7) {
		return 0xff, x
	}
	return 0, x
}

func (g *Game) sub_d721() (a, x byte) {
	R := &g.RAM
	v := g.addc(R[3], 7)
	R[0x12] = g.lsr4(v)
	R[0x12] = g.subc(0x0f, R[0x12])
	v = g.addc(R[4], 0x11) & 0xf0
	return g.sub_d765(g.addc(v, R[0x12]))
}

func (g *Game) sub_d744() (a, x byte) {
	R := &g.RAM
	v := g.addc(R[3], 7)
	R[0x12] = g.lsr4(v)
	R[0x12] = g.subc(0x0f, R[0x12])
	v = g.addc(R[4], 0x14) & 0xf0
	return g.sub_d765(g.addc(v, R[0x12]))
}

func (g *Game) sub_d765(idx byte) (a, x byte) {
	cell, x := g.mapCell(idx)
	if g.chain(cell, 0, 1, 2, 9, 0x0a, 0x0b) {
		return 0xff, x
	}
	return 0, x
}
