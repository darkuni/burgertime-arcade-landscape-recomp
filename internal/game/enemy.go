package game

// Enemy management: spawning (sub_dc2b), entering the screen (sub_dcbe),
// level speed ramp (sub_dd7a) and the chase logic (sub_de68).
//
// Per enemy x (0..5) the tables are:
//   $68+x  state: $ff free, bit 7 clear when active; $40/$20/$10 = falling / entering / stunned; low 2 bits = kind
//   $8a+x  frames left before the enemy may turn again
//   $72+4x.. pattern record: +0 mode, +1 delay, +2 -, +3 index into the pattern table

// sub_db8c sets up the enemy tables for a new level.
func (g *Game) sub_db8c() {
	R := &g.RAM
	y := byte(0)
	if g.Rd(0x4004)&0x08 == 0 {
		y = 6
	}
	x := uint16(g.addc(y, R[0x64]))
	R[0xbc] = g.Rom(0xdc05 + x)
	R[0xbd] = g.Rom(0xdc11 + x)
	R[0xbe] = g.Rom(0xdc1d + x)
	R[0xc2], R[0xc3], R[0xbf], R[0xc0], R[0xc1], R[0x90] = 0, 0, 0, 0, 0, 0
	lvl := R[0x61+uint16(R[zPlayer])] >> 2
	R[3] = lvl
	g.cmp(lvl, 8)
	if lvl >= 8 {
		lvl = 7
	}
	a := g.Rom(0xdbf3 + uint16(lvl))
	for i := uint16(0); i < 6; i++ {
		R[0x91+i] = a
	}
	a = g.Rom(0xdbfc + uint16(lvl))
	for i := uint16(0); i < 6; i++ {
		R[0xa1+i] = a
	}
	R[0x98] = g.Rom(0xdbf7)
	R[0xa8] = g.Rom(0xdc00)
}

// sub_dc2b brings a new enemy onto the screen when fewer than the level's quota are alive.
func (g *Game) sub_dc2b() {
	R := &g.RAM
	if R[0x6f]&0x80 != 0 || R[0x13]&0x1f != 0 {
		return
	}
	R[0xc2]++
	x := R[0xc2]
	if x == 3 {
		x = 0
	}
	R[0xc2] = x
	if R[0xbc+uint16(x)] == R[0xbf+uint16(x)] {
		return
	}
	y := byte(0)
	for R[0x68+uint16(y)] != 0xff {
		y++
		if y == 6 {
			return
		}
	}
	R[0xbf+uint16(x)]++
	x++
	R[0x68+uint16(y)] = x | 0x20
	R[0x0a] = y
	R[0x99+uint16(y)] = 1
	R[0xc3] = (R[0xc3] + 1) & 3
	x = R[0xc3] & 1
	R[0xa9+uint16(y)] = g.Rom(0xdc9f + uint16(x))
	yy := uint16(y) * 4
	g.Wr(0x1802+yy, g.Rom(0xdca1+uint16(x)))
	x = R[0x63]<<2 + R[0xc3]
	g.Wr(0x1803+yy, g.Rom(0xdca4+uint16(x)))
	g.sub_ddee()
}

// sub_ddee loads the next step of the movement pattern of enemy $0a.
func (g *Game) sub_ddee() {
	R := &g.RAM
	y := uint16(R[R[0x0a]+0x68]&3) * 2
	p3 := uint16(g.Rom(0xde2d+y)) | uint16(g.Rom(0xde2e+y))<<8
	p5 := uint16(g.Rom(0xde33+y)) | uint16(g.Rom(0xde34+y))<<8
	g.setPtr(3, p3)
	g.setPtr(5, p5)
	x := uint16(R[0x0a]) * 4
	i := uint16(R[0x75+x])
	R[0x72+x] = g.Rd(p3 + i)
	R[0x73+x] = g.Rd(p5 + i)
	R[0x74+x] = 0
	R[0x75+x]++
	if int8(g.Rd(p3+uint16(R[0x75+x]))) < 0 {
		R[0x75+x] = 0
	}
}

// Direction helpers of the chase logic: they point enemy $70 in a direction
// (keeping the low nibble of its direction byte in $71) and try to step.

func (g *Game) turn(dir byte) ret {
	R := &g.RAM
	x := R[0x70]
	R[0xa9+uint16(x)] = R[0x71]&0x0f | dir
	return g.sub_d4c6(x)
}

func (g *Game) sub_e15c() ret { return g.turn(0x60) }
func (g *Game) sub_e16d() ret { return g.turn(0x80) }
func (g *Game) sub_e17e() ret { return g.turn(0x20) }
func (g *Game) sub_e18f() ret { return g.turn(0x40) }

// sub_e1a0: Z set when the second coordinate lies in the right-hand columns ($c..$f of the cell).
func (g *Game) sub_e1a0() bool {
	a := g.RAM[4] & 0x0f
	g.carry = a >= 0x0c
	return a >= 0x0c
}

// sub_e1b9: Z set when the first coordinate (+2) is on a floor line; returns the line number in a.
func (g *Game) sub_e1b9() (a byte, z bool) {
	R := &g.RAM
	R[8] = R[3] + 2
	switch R[8] & 0x0c {
	case 0:
		hi := R[8] & 0xf0
		return hi, g.chain(hi, 0x30, 0x60, 0x90, 0xc0)
	case 8:
		hi := R[8] & 0xf0
		return hi, g.chain(hi, 0x10, 0x40, 0x70, 0xa0, 0xd0)
	}
	g.carry = true
	return R[8] & 0x0c, false
}

// sub_dcbe moves the enemies that are still walking onto the screen.
func (g *Game) sub_dcbe() {
	R := &g.RAM
	if R[0x6f]&0x80 != 0 {
		return
	}
	for x := byte(0); x != 6; x++ {
		s := R[0x68+uint16(x)]
		if s&0xd0 != 0 || s&0x20 == 0 {
			continue
		}
		R[0x70] = x
		R[0x71] = R[0xa9+uint16(x)]
		y := uint16(x) * 4
		R[3] = g.Rd(0x1802 + y)
		R[4] = g.Rd(0x1803 + y)
		g.enterStep(x)
	}
}

func (g *Game) enterStep(x byte) {
	R := &g.RAM
	set := func(d byte) {
		R[0xa9+uint16(x)] = R[0xa9+uint16(x)]&0x0f | d
	}
	switch {
	case R[3] >= 0xd9:
		set(0x20)
	case R[3] < 0x18:
		set(0x40)
	default:
		a, z := g.sub_e1b9()
		if !z {
			if R[3] >= 0x80 {
				set(0x40)
			} else {
				set(0x20)
			}
			break
		}
		switch a {
		case 0x10, 0x40, 0x70, 0xa0, 0xd0:
			ok := false
			if g.sub_e1a0() {
				if g.sub_e17e().z || g.sub_e18f().z {
					ok = true
				}
			}
			if !ok && g.sub_e15c().z {
				ok = true
			}
			if ok {
				R[0x68+uint16(R[0x70])] &= 0x0f
			}
		default:
			if R[3] >= 0x80 {
				set(0x40)
			} else {
				set(0x20)
			}
		}
	}
	g.sub_d36c(R[0x70])
}

// sub_dd7a ramps the speed of the enemies up as the level goes on and
// advances the movement patterns.
func (g *Game) sub_dd7a() {
	R := &g.RAM
	if R[0x13]&0x3f != 0 {
		return
	}
	R[0x90]++
	if g.cmpGE(R[0x90], 0x14) {
		R[0x90] = 0
		for x := 5; x >= 0; x-- {
			if R[0x91+x]&0x10 != 0 {
				if R[0xa1+x] == 2 {
					continue
				}
				R[0xa1+x]--
				R[0x91+x] = 3
			} else {
				R[0x91+x] = R[0x91+x]<<1 | 1
			}
		}
	}
	y, x := byte(5), byte(0x14)
	for {
		if R[0x68+uint16(y)]&0x80 == 0 {
			if R[0x73+uint16(x)] != 0 {
				R[0x73+uint16(x)]--
			} else {
				R[0x0a] = y
				g.sub_ddee()
			}
		}
		y--
		x -= 4
		if int8(x) < 0 {
			return
		}
	}
}

// sub_de68 steers every active enemy: it chases the chef.
func (g *Game) sub_de68() {
	R := &g.RAM
	if R[0x6f]&0x80 != 0 {
		return
	}
	for R[0x70] = 5; int8(R[0x70]) >= 0; R[0x70]-- {
		x := R[0x70]
		if R[0x68+uint16(x)]&0xf0 != 0 {
			continue
		}
		g.chase(x)
	}
}

// Preferred direction orders of the chase logic (2 = up, 4 = down, 6 = left, 8 = right in the
// original's direction nibble), by the original's branch target.
var chaseOrder = map[uint16][4]byte{
	0xdfea: {6, 2, 4, 8}, 0xe004: {2, 6, 8, 4}, 0xe01e: {8, 2, 4, 6}, 0xe038: {6, 4, 8, 2},
	0xe058: {4, 8, 6, 2}, 0xe072: {8, 4, 6, 2}, 0xe08c: {2, 6, 4, 8}, 0xe0a6: {6, 2, 4, 8},
	0xe0c0: {4, 6, 2, 8}, 0xe0da: {2, 8, 4, 6}, 0xe0f4: {8, 4, 2, 6}, 0xe10e: {4, 8, 2, 6},
}

func (g *Game) chase(x byte) {
	R := &g.RAM
	y := uint16(x) * 4
	R[3], R[4] = g.Rd(0x1802+y), g.Rd(0x1803+y)
	if R[0x8a+uint16(x)] != 0 {
		R[0x8a+uint16(x)]--
		g.follow(x)
		return
	}
	if !g.sub_e1a0() {
		g.follow(x)
		return
	}
	if _, z := g.sub_e1b9(); !z {
		g.follow(x)
		return
	}
	R[0x17], R[0x18] = g.Rd(0x181e), g.Rd(0x181f)
	switch mode := R[0x72+y]; mode {
	case 3:
		R[0x17] += 8
	case 0, 2, 4:
	default:
		R[0x18] += 8
	}
	R[0x71] = R[0xa9+uint16(x)]
	var target uint16
	switch R[0x71] & 0xf0 {
	case 0x20:
		if R[4]&0xf8 >= R[0x18]&0xf8 {
			target = 0xdfea
		} else {
			target = 0xe01e
		}
	case 0x40:
		if R[4]&0xf8 >= R[0x18]&0xf8 {
			target = 0xe038
		} else {
			target = 0xe072
		}
	case 0x60:
		a, b := R[3]&0xf8, R[0x17]&0xf8
		switch {
		case a < b:
			target = 0xe0c0
		case a == b:
			target = 0xe0a6
		default:
			target = 0xe08c
		}
	default:
		a, b := R[3]&0xf8, R[0x17]&0xf8
		switch {
		case a < b:
			target = 0xe10e
		case a == b:
			target = 0xe0f4
		default:
			target = 0xe0da
		}
	}
	g.tryDirs(x, target)
}

// follow keeps the enemy going the way it was going (dfbe).
func (g *Game) follow(x byte) {
	R := &g.RAM
	if g.sub_d4c6(x).z {
		g.sub_d36c(x)
		return
	}
	R[0x71] = R[0xa9+uint16(x)]
	var target uint16
	switch R[0x71] & 0xf0 {
	case 0x20:
		target = 0xe004
	case 0x40:
		target = 0xe058
	case 0x60:
		target = 0xe0a6
	default:
		target = 0xe0f4
	}
	g.tryDirs(x, target)
}

// tryDirs tries the four directions in the order the branch target dictates; the
// enemy is removed when none works.
func (g *Game) tryDirs(x byte, target uint16) {
	R := &g.RAM
	for _, d := range chaseOrder[target] {
		var r ret
		switch d {
		case 6:
			r = g.sub_e15c()
		case 8:
			r = g.sub_e16d()
		case 2:
			r = g.sub_e17e()
		default:
			r = g.sub_e18f()
		}
		if r.z {
			// e128
			R[0x74+uint16(R[0x70])] = 0
			R[0x8a+uint16(R[0x70])] = 0x10
			g.sub_d36c(R[0x70])
			return
		}
	}
	// e140: nothing works - the enemy leaves
	kind := R[0x68+uint16(R[0x70])] & 3
	R[0xbe+uint16(kind)]--
	g.Wr(0x1800+uint16(kind)*4, 0)
	R[0x68+uint16(R[0x70])] = 0xff
}
