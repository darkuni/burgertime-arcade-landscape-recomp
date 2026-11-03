package game

// Walking over the burger ingredients (sub_d99b and helpers).
//
// The maze is stored in video RAM ($1000..) with one flag byte per tile in colour
// RAM ($1400..). The chef flattens the tiles of an ingredient he walks over and,
// when a whole row has been stepped on, the ingredient starts to fall.

// sub_db6d moves the tile at ($03)+y and its lower neighbour on to their pressed variants.
func (g *Game) sub_db6d(y byte) {
	p := g.ptr(3)
	g.RAM[0x12] = y
	g.Wr(p+uint16(y), g.Rd(p+uint16(y))+4)
	y2 := y + 0x20
	g.Wr(p+uint16(y2), g.Rd(p+uint16(y2))+4)
}

// sub_db5b tests the tile at offset y: returns x (0xff when the tile has no
// ingredient flag, otherwise 0) and z set when nothing was pressed.
func (g *Game) sub_db5b(y byte) (x byte, z bool) {
	x = 0xff
	if g.Rd(g.ptr(5)+uint16(y))&3 == 0 {
		return x, true
	}
	x = 0
	if g.Rd(g.ptr(3)+uint16(y))&0x1c == 0 {
		return x, true
	}
	g.sub_db6d(y)
	return x, false
}

// sub_d99b checks whether the chef stands on an ingredient.
func (g *Game) sub_d99b() {
	R := &g.RAM
	if R[0xba] == 6 || R[0xba] == 8 {
		return
	}
	a := ^g.Rd(0x181e)
	g.carry = true
	a = g.sbc(a, 0x28)
	R[3] = a >> 3
	R[4] = 0
	v := g.addc(g.Rd(0x181f)&0xf8, 0x10)
	c := v >> 7
	v <<= 1
	R[4] = c
	c = v >> 7
	v <<= 1
	R[4] = R[4]<<1 | c
	v = g.addc(v, R[3])
	R[5], R[3] = v, v
	R[4] = g.adc(R[4], 0x10)
	R[5+1] = g.addc(R[4], 4)
	// pointers: ($03) -> tile codes, ($05) -> flags
	p3, p5 := g.ptr(3), g.ptr(5)
	y := byte(0x24)
	if g.Rd(p5+uint16(y))&3 == 0 {
		return
	}
	y = 4
	if g.Rd(p5+uint16(y))&3 == 0 {
		return
	}
	if g.Rd(p3+uint16(y))&0x1f >= 4 {
		return
	}
	g.sub_eb61(8)
	g.sub_db6d(y)
	var x byte
	if R[0xba] != 2 {
		y++
		for {
			var z bool
			x, z = g.sub_db5b(y)
			if z {
				// da2e
				if x == 0 {
					for {
						y++
						if g.Rd(p5+uint16(y))&3 == 0 {
							break
						}
					}
				}
				x = 4
				for {
					y--
					if g.Rd(p3+uint16(y))&0x1c == 0 {
						return
					}
					x--
					if x == 0 {
						break
					}
				}
				break
			}
			y++
			if y == 8 {
				y = 4
				break
			}
		}
	} else {
		y--
		for {
			var z bool
			x, z = g.sub_db5b(y)
			if z {
				// da52
				if x == 0 {
					for {
						y--
						if g.Rd(p5+uint16(y))&3 == 0 {
							break
						}
					}
				}
				x = 4
				for {
					y++
					if g.Rd(p3+uint16(y))&0x1c == 0 {
						return
					}
					x--
					if x == 0 {
						break
					}
				}
				y -= 3
				break
			}
			y--
			if y == 0 {
				y = 1
				break
			}
		}
	}
	// da77: the row has been stepped on: find the ingredient and start it falling
	g.sub_eb61(9)
	sum := uint16(y) + uint16(R[3])
	R[7] = byte(sum)
	hi := (uint16(R[4]) + sum>>8) & 3
	R[8] = byte(hi)
	R[9] = (R[7] & 0x1f) << 3
	R[7] = byte((hi<<8|uint16(R[7]))>>2) & 0xf8
	R[8] = byte(hi >> 2)
	for i := uint16(0); ; i += 4 {
		t := R[0x202+i]
		if t == 0 {
			return
		}
		if (^R[0x203+i])&0xf8 != R[9] || R[0x204+i]&0xf8 != R[7] || t >= 0x0f {
			continue
		}
		R[0x202+i] = t&0x0f | 0x10
		R[0x204+i] = R[0x204+i]&0xfc | 4
		R[0x205+i] = 1
		for k := byte(0); k < 6; k++ {
			R[0x0a] = k
			if R[0x68+uint16(k)]&0xa0 != 0 {
				continue
			}
			yy := uint16(k) * 4
			a := g.addc(g.Rd(0x1802+yy), 8)
			if a >= R[0x203+i] {
				continue
			}
			a += 0x20
			if a < R[0x203+i] {
				continue
			}
			b := g.addc(g.Rd(0x1803+yy)&0xf8, 0x10)
			R[0x0b] = b
			if R[0x204+i]&0xf8 != b {
				continue
			}
			R[0x68+uint16(k)] = R[0x68+uint16(k)]&0x0f | 0x40
			R[0xb1+uint16(k)] = byte(i)
			R[0x205+i] += 2
			g.sub_eb61(0x12)
		}
		R[0x0a] = 6
		return
	}
}
