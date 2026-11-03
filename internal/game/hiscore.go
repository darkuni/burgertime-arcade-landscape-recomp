package game

// High score table and initials entry (f1af..f81a).
// The routines keep their working variables at the original zero page addresses.

const (
	zScoreTmp  = 0xe7 // $e7..$e9: table entry being compared
	zRank      = 0xed
	zCurPos    = 0xee // position of the cursor in the letter grid
	zLastDir   = 0xef
	zEntState  = 0xe2
	zEntFlags  = 0xe3
	zEntDelay  = 0xe0
	zBtnDelay  = 0xe1
	zEntPlayer = 0xcb
)

func (g *Game) ptrAt(z byte) uint16 { return g.ptr(z) }

// sub_f1af: checks the score in $cc..$ce against the table and lets the player enter initials.
func (g *Game) sub_f1af() {
	R := &g.RAM
	R[0xf3] = 0
	R[zRank] = 0
	R[0xe4], R[0xe5] = 0x36, 0
	for {
		p := g.ptr(0xe4)
		for y := 2; y >= 0; y-- {
			R[zScoreTmp+y] = R[p+uint16(y)]
		}
		if R[0xe7] == 0xff {
			return
		}
		lo := int(R[0xe4]) + 3
		R[0xe4] = byte(lo)
		if lo > 0xff {
			R[0xe5]++
		}
		R[zRank]++
		// entry >= score (BCD, least significant byte first)?
		if bcdGE(R[0xe9], R[0xe8], R[0xe7], R[0xce], R[0xcd], R[0xcc]) {
			continue
		}
		break
	}
	x := R[zRank]
	R[0xd8] = g.Rom(0xf23a+uint16(x)) + 0x36
	R[0xd9] = 0
	R[0xda] = 2
	g.sub_f248()
	R[0xcc], R[0xcd], R[0xce] = 0, 0, 0
	R[0xd8] = g.Rom(0xf240+uint16(x)) + 0x36
	R[0xd9] = 0
	R[0xda] = 2
	g.sub_f248()
	g.sub_f28d()
	g.sub_ccbd()
}

func bcdGE(a2, a1, a0, b2, b1, b0 byte) bool {
	if a2 != b2 {
		return a2 > b2
	}
	if a1 != b1 {
		return a1 > b1
	}
	return a0 >= b0
}

// sub_f248: inserts the record in $cc.. at ($d8) and shifts the following records down (the last one drops off).
func (g *Game) sub_f248() {
	R := &g.RAM
	for {
		da := R[0xda]
		p := g.ptr(0xd8)
		for y := int(da); y >= 0; y-- {
			R[0xea+y] = R[p+uint16(y)]
		}
		if R[0xea] == 0xff {
			return
		}
		for y := int(da); y >= 0; y-- {
			R[p+uint16(y)] = R[0xcc+y]
		}
		for x := int(da); x >= 0; x-- {
			R[0xcc+x] = R[0xea+x]
		}
		sum := int(R[0xd8]) + int(da) + 1
		R[0xd8] = byte(sum)
		if sum > 0xff {
			R[0xd9]++
		}
	}
}

// sub_f28d: the "enter your initials" screen.
func (g *Game) sub_f28d() {
	R := &g.RAM
	g.sub_c976()
	g.sub_f398()
	g.sub_f4b7()
	for R[0xdc] = 5; R[0xdc]&0x80 == 0; R[0xdc]-- {
		dd := R[0xdc] + 3
		R[0xdd] = dd
		g.drawScoreAt(uint16(R[0xdc]*2+0x10), dd*2+dd)
	}
	R[0xdc] = 4
	g.sub_f372()
	R[0xde], R[0xdf], R[0xe0] = 0, 7, 6
	R[0xe1], R[0xe2], R[0xe3] = 0, 0, 0
	for {
		g.sub_f474()
		g.sub_f490()
		if R[zEntState]&0x0f == 0x0f {
			g.sub_f474()
			g.Wr(0x4003, 0)
			return
		}
		if R[zEntState]&0x80 != 0 && R[zEntFlags]&0x80 == 0 {
			R[0xde] = 0xff
			R[zEntFlags] |= 0x80
			R[0xdf] = 0
		}
		if R[0xde]|R[0xdf] == 0 {
			if R[zEntState]&0x80 != 0 {
				R[zEntState] |= 0x0f
			}
			R[zEntState] |= 0x80
		}
		g.sub_f4ee()
		if g.sub_f53b() {
			return
		}
		g.sub_f5a8()
		g.sub_f5ee()
		g.sub_f75b()
		if R[zEntState]&0x20 != 0 {
			g.sub_f398()
			g.sub_f372()
			g.sub_f4b7()
			x := R[zEntState] ^ 0x20
			if R[zEntFlags]&0x40 == 0 {
				x++
			}
			R[zEntState] = x
		}
		g.sub_f7fd()
		R[zEntFlags] &= 0xbf
		if R[zEntState]&0x0f >= 3 {
			R[zEntFlags] |= 0x20
		}
	}
}

// sub_f372: sets up the cursor sprite.
func (g *Game) sub_f372() {
	g.Wr(0x1802, 0xb4)
	g.Wr(0x1803, 0x40)
	g.Wr(0x1801, 0x47)
	g.Wr(0x1800, 1)
	g.Wr(0x1804, 0)
	g.RAM[zCurPos], g.RAM[zLastDir] = 0, 0
	g.Wr(0x1805, 0x17)
}

// sub_f398: copies the screen furniture of the entry screen from the ROM.
func (g *Game) sub_f398() {
	R := &g.RAM
	for dd := 8; dd >= 0; dd-- {
		R[0xdd] = byte(dd)
		x := uint16(dd) * 2
		R[0xd8] = g.Rom(0xf3ce + x)
		R[0xda] = g.Rom(0xf3e0 + x)
		R[0xd9] = g.Rom(0xf3ce + x + 1)
		R[0xdb] = g.Rom(0xf3e0 + x + 1)
		src, dst := g.ptr(0xd8), g.ptr(0xda)
		for y := int(g.Rom(0xf3f2 + uint16(dd))); y >= 0; y-- {
			g.Wr(dst+uint16(y), g.Rd(src+uint16(y)))
		}
	}
	R[0xdd] = 0xff
}

// sub_f474: one frame of the entry screen's timing.
func (g *Game) sub_f474() { g.vblankWork() }

// sub_f490: counts the timer and the delay counters down.
func (g *Game) sub_f490() {
	R := &g.RAM
	v := int(R[0xde]) | int(R[0xdf])<<8
	v = (v - 1) & 0xffff
	R[0xde], R[0xdf] = byte(v), byte(v>>8)
	R[0xe0]--
	if R[0xe0]&0x80 != 0 {
		R[0xe0] = 4
	}
	R[0xe1]--
	if R[0xe1]&0x80 != 0 {
		R[0xe1] = 0
	}
}

// sub_f4b7: draws the three letter names of the table.
func (g *Game) sub_f4b7() {
	R := &g.RAM
	R[0xd8], R[0xd9] = 0x8b, 0x12
	x := 0
	for {
		for y := 0; y < 3; y++ {
			g.Wr(g.ptr(0xd8)+uint16(y), R[0x48+x])
			x++
			if x == 0x0f {
				return
			}
		}
		lo := int(R[0xd8]) + 0x40
		R[0xd8] = byte(lo)
		if lo > 0xff {
			R[0xd9]++
		}
	}
}

// sub_f4ee: blinks the cursor.
func (g *Game) sub_f4ee() {
	R := &g.RAM
	if R[zEntDelay] != 0 {
		return
	}
	g.Wr(0x1801, (R[zEntState]&0x10)>>4+0x43)
	if R[zEntState]&0x80 != 0 {
		R[0xdc]--
		a := byte(0x47)
		if R[0xdc] == 0 {
			R[0xf3]++
			if R[0xf3] == 1 {
				g.Wr(0x4003, 0x1e)
			}
			R[0xdc] = 3
			a = 0x4c
		}
		g.Wr(0x1801, a)
	}
	R[zEntState] ^= 0x10
}

// sub_f53b: joystick; returns true when a start button ends the entry.
func (g *Game) sub_f53b() (abort bool) {
	R := &g.RAM
	if (g.Rd(0x4002)^0xff)&3 != 0 {
		return true
	}
	if R[zEntDelay] != 0 || R[zEntState]&0xe0 != 0 {
		return false
	}
	x := uint16(R[zEntPlayer])
	if g.Rd(0x4003)&0x40 == 0 {
		x = 0
	}
	d := g.Rom(0xf594 + uint16((g.Rd(0x4000+x)^0xff)&0x0f))
	if d == R[zLastDir] {
		return false
	}
	R[zLastDir] = d
	a := d + R[zCurPos]
	if a&0x80 != 0 {
		a = 0
	}
	if a >= 0x24 {
		a = 0x23
	}
	R[zCurPos] = a
	g.Wr(0x4003, 0x15)
	return false
}

// sub_f5a8: places the cursor sprite on the letter grid.
func (g *Game) sub_f5a8() {
	R := &g.RAM
	if R[zEntState]&0xe0 != 0 {
		return
	}
	p := R[zCurPos]
	x := byte(0x30)
	switch {
	case p >= 0x1e:
		x = 0x78
	case p >= 0x14:
		x = 0x60
	case p >= 0x0a:
		x = 0x48
	}
	g.Wr(0x1803, x)
	g.Wr(0x1802, g.Rom(0xf5e2+uint16(p%10)))
}

// sub_f5ee: the fire button selects a letter, or one of the END / RUB / SPACE keys.
func (g *Game) sub_f5ee() {
	R := &g.RAM
	if R[zBtnDelay] != 0 || R[zEntState]&0xe0 != 0 {
		return
	}
	x := uint16(R[zEntPlayer])
	if g.Rd(0x4003)&0x40 == 0 {
		x = 0
	}
	if (g.Rd(0x4000+x)^0xff)&0x10 == 0 {
		return
	}
	if R[zEntState]&3 != 3 {
		g.Wr(0x4003, 0x11)
	}
	pos := R[zCurPos]
	if pos >= 0x1e {
		i := uint16(pos-0x1e) * 2
		R[0xd8], R[0xd9] = g.Rom(0xf6aa+i), g.Rom(0xf6ab+i)
		R[zBtnDelay] = 0x3f
		g.Wr(0x4003, 0x17)
		switch uint16(R[0xd8]) | uint16(R[0xd9])<<8 {
		case 0xf6b9:
			g.sub_f6b9()
		case 0xf6d5:
			g.sub_f6d5()
		case 0xf6cb:
			g.sub_f6cb()
		default:
			panic("bad jump table")
		}
		return
	}
	if R[zEntFlags]&0x20 != 0 {
		return
	}
	R[0xd9] = 0x60
	px := int(pos)
	if px >= 0x1a && px < 0x1e {
		px = px - 0x19
		R[0xd8], R[0xd9] = 0x50, 0x66
	} else {
		R[0xd8] = 0x50
	}
	for ; px >= 0; px-- {
		lo := int(R[0xd8]) + 8
		R[0xd8] = byte(lo)
		if lo > 0xff {
			R[0xd9]++
		}
	}
	R[0xda], R[0xdb] = 0xe0, 0x62
	g.sub_f70a()
	g.sub_f6f4()
	g.Wr(0x1804, 1)
	g.sub_f730()
	g.Wr(g.ptr(0xd8), 0)
	R[zEntState] |= 0x40
}

// sub_f6b9 / f6cb / f6d5: the three special keys.
func (g *Game) sub_f6b9() {
	if g.RAM[zEntFlags]&0x20 != 0 {
		return
	}
	g.RAM[zCurPos] = 0xf5
	g.sub_f7c8(g.RAM[zRank])
}

func (g *Game) sub_f6cb() { g.RAM[zEntState] |= 0x80 }

func (g *Game) sub_f6d5() {
	R := &g.RAM
	if R[zEntState]&0x0f == 0 {
		return
	}
	R[zEntState]--
	R[zCurPos] = 0xf5
	g.sub_f7c8(R[zRank])
	R[zEntFlags] = (R[zEntFlags] | 0x40) &^ 0x20
}

// sub_f6f4: places the "shadow" sprite relative to the cursor.
func (g *Game) sub_f6f4() {
	g.Wr(0x1806, g.Rd(0x1802)-4)
	g.Wr(0x1807, g.Rd(0x1803)-8)
}

// sub_f70a: copies three rows of eight bytes from ($d8) to ($da).
func (g *Game) sub_f70a() {
	R := &g.RAM
	for x := 2; x >= 0; x-- {
		s, d := g.ptr(0xd8), g.ptr(0xda)
		for y := 7; y >= 0; y-- {
			g.Wr(d+uint16(y), g.Rd(s+uint16(y)))
		}
		R[0xd9] += 0x20
		R[0xdb] += 0x20
	}
}

// sub_f730: converts the shadow sprite's position into a character cell address in $d8.
func (g *Game) sub_f730() {
	R := &g.RAM
	R[0xd9] = g.Rd(0x1807)
	R[0xd8] = g.Rd(0x1806) ^ 0xff
	lsr := func(z byte) (c byte) {
		c = R[z] & 1
		R[z] >>= 1
		return
	}
	ror := func(z byte, c byte) byte {
		n := R[z] & 1
		R[z] = R[z]>>1 | c<<7
		return n
	}
	lsr(0xd9)
	lsr(0xd9)
	lsr(0xd9)
	c := lsr(0xd9)
	c = ror(0xd8, c)
	c = lsr(0xd9)
	c = ror(0xd8, c)
	c = lsr(0xd9)
	ror(0xd8, c)
	R[0xd9] += 0x10
	R[0xd8]--
}

// sub_f75b: moves the cursor sprite towards the letter slot that is being filled.
func (g *Game) sub_f75b() {
	R := &g.RAM
	if R[zEntState]&0xe0 != 0x40 {
		return
	}
	d8 := byte(0x9b)
	for x := R[zEntState] & 0x0f; x > 0; x-- {
		d8 -= 8
	}
	R[0xd8] = d8
	R[0xd9] = g.Rom(0xf7bf + uint16(R[zRank]))
	v := g.Rd(0x1802)
	switch {
	case d8 == v:
	case d8 < v:
		g.Wr(0x1802, v-1)
	default:
		g.Wr(0x1802, v+1)
	}
	t := g.Rd(0x1803)
	if R[0xd9] == t {
		if R[0xd8] == g.Rd(0x1802) {
			g.sub_f7c8(R[zRank])
			return
		}
	} else {
		g.Wr(0x1803, t+1)
	}
	g.sub_f6f4()
}

// sub_f7c8: stores the selected letter in the name being entered.
func (g *Game) sub_f7c8(x byte) {
	R := &g.RAM
	R[zEntState] = (R[zEntState] | 0x20) &^ 0x40
	R[0xd8] = g.Rom(0xf240+uint16(x)) + 0x36
	R[0xd9] = 0
	y := uint16(R[zEntState] & 0x0f)
	a := R[zCurPos]
	if a >= 0x1a && a < 0x1e {
		a += 0xb2
	} else {
		a += 0x0b
	}
	R[g.ptr(0xd8)+y] = a
}

// sub_f7fd: shows the "end of entry" marker.
func (g *Game) sub_f7fd() {
	if g.RAM[zEntState]&0x80 == 0 {
		return
	}
	g.Wr(0x1802, 0xd0)
	g.Wr(0x1803, g.Rom(0xf7bf+uint16(g.RAM[zRank]))-0x0c)
	g.Wr(0x1804, 0)
}
