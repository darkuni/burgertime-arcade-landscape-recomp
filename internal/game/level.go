package game

// Level set-up and the attract mode demos (c818, c837, c999, c9b9, cb2a, cb9e, cbf8, c552, c62c).

// sub_c818: start of a new game for the current player: level 1, no burger pieces.
func (g *Game) sub_c818() {
	g.RAM[0x61], g.RAM[0x62] = 1, 1
	g.sub_eb61(4)
	g.RAM[0x65], g.RAM[0x66] = 0, 0
	g.RAM[zPlayer] = 0
	g.sub_cce9()
	g.sub_c333()
	g.RAM[0x61]--
}

// sub_c837: sets up the level of the current player (screen, objects, chef, enemies).
func (g *Game) sub_c837() {
	R := &g.RAM
	g.sub_c976()
	if R[zNewLife] == 0 {
		g.sub_eb61(4)
		if g.Rd(0x4004)&0x10 == 0 && R[zPlaying] != 0 {
			x := R[zPlayer]
			g.carry = false
			R[zPepper+x] = g.bcdAdd(R[zPepper+x], 1)
		}
		x := R[zPlayer]
		R[0x61+x]++
		R[0x65+x] = 0
		g.sub_cce9()
	}
	R[zNewLife] = 0
	g.drawBlockList()
	x := R[zPlayer]
	y := int(R[0x61+x]) - 1
	for y >= 6 {
		y -= 6
	}
	R[0x63] = byte(y)
	y = int(R[0x61+x]) - 1
	if y >= 6 {
		y = 5
	}
	R[0x64] = byte(y)
	l := uint16(R[0x63])
	g.Wr(0x181c, g.Rom(0xcec9+l))
	g.Wr(0x181d, g.Rom(0xced1+l))
	g.Wr(0x181e, g.Rom(0xced9+l))
	g.Wr(0x181f, g.Rom(0xcee1+l))
	for i := 0; i < 8; i++ {
		R[0x68+i] = 0xff
		R[0x99+i] = 1
		R[0xa1+i] = 4
		R[0xa9+i] = 0
	}
	R[0x6f] = 0
	R[0xa8] = 3
	R[0xc5], R[0xc4] = 0, 0
	g.sub_db8c()
	g.sub_c999()
	if R[zTwoPlay] != 0 {
		g.drawScore(R[zPlayer] ^ 1)
	}
	g.drawScore(R[zPlayer])
	g.drawScore(2)
	g.sub_cb2a()
	g.sub_cb6e()
	g.sub_cb9e(R[zPlayer])
	g.sub_cbf8()
	R[zFrame] = 0x3e
	if R[zPlaying] != 0 {
		return
	}
	// demo: three enemies and the burger logo chef
	R[0x68], R[0x69], R[0x6a] = 1, 2, 3
	g.Wr(0x1802, 0x90)
	g.Wr(0x1803, 0x4d)
	g.Wr(0x1806, 0x60)
	g.Wr(0x1807, 0x3d)
	g.Wr(0x180a, 0x30)
	g.Wr(0x180b, 0x3d)
	R[0xa9], R[0xaa], R[0xab] = 0x80, 0x80, 0x80
	R[0xb0] = 0x40
	g.Wr(0x181e, 0x18)
	g.Wr(0x181f, 0x1d)
	R[0xa1], R[0xa2], R[0xa3] = 0xff, 0xff, 0xff
}

// sub_c999: the "1UP", "HI-SCORE" (and "2UP") captions.
func (g *Game) sub_c999() {
	g.printText(0xca08)
	g.printText(0xc9fc)
	if g.RAM[zTwoPlay] != 0 {
		g.printText(0xca13)
	}
}

// sub_c9b9: makes the caption of the active player blink in a two player game.
func (g *Game) sub_c9b9() {
	if g.RAM[zTwoPlay] == 0 {
		return
	}
	a := g.RAM[zFrame] & 0x3f
	if a == 0 {
		if g.RAM[zPlayer] == 0 {
			g.printText(0xca02)
		} else {
			g.printText(0xca19)
		}
		return
	}
	g.carry = a >= 0x0f
	if a != 0x0f {
		return
	}
	if g.RAM[zPlayer] == 0 {
		g.printText(0xc9fc)
	} else {
		g.printText(0xca13)
	}
}

// sub_cb2a: draws the remaining lives (burger-chef icons) of the current player.
func (g *Game) sub_cb2a() {
	lives := g.RAM[0x29+g.RAM[zPlayer]]
	f3 := 0x5c - lives
	y := lives
	if lives >= 9 {
		y = 9
		f3 = 0x53
	} else {
		for i := uint16(0x10); i >= 1; i-- {
			g.Wr(0x184c+i, 0)
		}
		if lives == 0 {
			g.RAM[0xf3], g.RAM[0xf4] = f3, 0x18
			return
		}
	}
	g.RAM[0xf3], g.RAM[0xf4] = f3, 0x18
	p := uint16(f3) | 0x1800
	for ; y > 0; y-- {
		g.Wr(p+uint16(y), 0xc8)
	}
}

// sub_cb9e: draws the level number of player x as flags (10, 5, 1).
func (g *Game) sub_cb9e(x byte) {
	n := g.RAM[0x61+x]
	p := uint16(g.Rom(0xcbf4)) | uint16(g.Rom(0xcbf5))<<8
	y := byte(0x80)
	for i := 0; i < 5; i++ {
		var c byte
		switch {
		case n >= 10:
			n -= 10
			c = 0xc6
		case n >= 5:
			n -= 5
			c = 0xc5
		case n >= 1:
			n--
			c = 0xc4
		default:
			return
		}
		g.Wr(p+uint16(y), c)
		y -= 0x20
	}
}

// sub_cbf8: unpacks the maze of the level into the top nibbles of $0400.. and selects the background page.
func (g *Game) sub_cbf8() {
	R := &g.RAM
	g.sub_ccbd()
	src := uint16(g.Rom(0xceb1+uint16(R[0x63])*2)) | uint16(g.Rom(0xceb2+uint16(R[0x63])*2))<<8
	dst := uint16(0x410)
	x := 0x68
	for {
		for {
			b := g.Rom(src)
			R[dst] |= b & 0xf0
			dst++
			R[dst] |= b << 4
			dst++
			src++
			x--
			if dst&7 == 0 {
				break
			}
		}
		dst += 0x7f
		for {
			b := g.Rom(src)
			R[dst] |= b & 0xf0
			dst--
			R[dst] |= b << 4
			dst--
			src++
			x--
			if dst&7 == 7 {
				break
			}
		}
		dst -= 0x77
		if x == 0 {
			break
		}
	}
	f3 := R[0x63] & 0x0f
	R[0xf3] = f3
	v := f3
	if g.Rd(0x4003)&0x40 == 0 || R[zPlayer] == 0 {
		v = g.Rom(0xccb3 + uint16(f3))
	}
	g.Wr(0x4004, v|0x10)
}

// sub_c552: the attract mode demo with the chef walking into the enemies.
func (g *Game) sub_c552() {
	R := &g.RAM
	R[0x19] = 0
	R[zInput] = 1
	g.sub_c818()
	g.sub_c837()
	R[0x68], R[0x69], R[0x6a] = 1, 2, 3
	g.Wr(0x1802, 0x90)
	g.Wr(0x1803, 0x4d)
	g.Wr(0x1806, 0x60)
	g.Wr(0x1807, 0x3d)
	g.Wr(0x180a, 0x30)
	g.Wr(0x180b, 0x3d)
	R[0xa9], R[0xaa], R[0xab], R[0xb0] = 0x40, 0x40, 0x40, 0x40
	R[0xa8], R[0xba] = 4, 4
	g.Wr(0x181e, 0x18)
	g.Wr(0x181f, 0x1d)
	g.sub_de68()
	for {
		g.waitFrames(1)
		R[zFrame]++
		if r := g.sub_d4c6(7); !r.z {
			return
		}
		g.sub_d36c(7)
		g.sub_d99b()
		g.sub_ebad()
		g.sub_e293()
	}
}

// sub_c62c: the second demo, in which the chef is chased by six enemies and pepper is used.
func (g *Game) sub_c62c() {
	R := &g.RAM
	g.sub_c818()
	g.sub_c837()
	R[0x68], R[0x69], R[0x6a], R[0x6b], R[0x6c], R[0x6d] = 1, 1, 2, 2, 3, 3
	for _, a := range []uint16{0x1803, 0x1807, 0x180b, 0x180f, 0x1813, 0x1817, 0x181f} {
		g.Wr(a, 0x1d)
	}
	for a := 0xa9; a <= 0xae; a++ {
		R[a] = 0x40
	}
	g.Wr(0x1802, 0x89)
	g.Wr(0x1806, 0x59)
	g.Wr(0x180a, 0x60)
	g.Wr(0x180e, 0x90)
	g.Wr(0x1812, 0x97)
	g.Wr(0x1816, 0x30)
	g.Wr(0x181e, 0x18)
	R[zFrame], R[zFrame16L] = 1, 1
	g.sub_de68()
	for {
		g.waitFrames(1)
		R[zFrame]++
		if R[zFrame] == 0 {
			R[zFrame16L]++
		}
		r := g.sub_d4c6(7)
		if !r.z {
			if R[zFrame16L] >= 4 {
				g.printText(0xc700)
				g.waitFrames(0xff)
				g.waitFrames(0x40)
				g.waitFrames(0xa0)
				return
			}
		} else if R[0x6e]&0x80 != 0 {
			g.sub_d36c(7)
			switch g.Rd(0x181e) {
			case 0x22, 0x4a, 0x82:
				g.sub_d2b0()
				g.Wr(0x181e, g.Rd(0x181e)+2)
			}
		}
		g.sub_d899()
		g.sub_d99b()
		g.sub_e7e2()
		g.sub_e293()
	}
}
