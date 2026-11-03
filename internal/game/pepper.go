package game

// The pepper: its animation (sub_d899) and hits on enemies (sub_e775).

// sub_e775 checks the pepper spray (sprite at $181a/$181b) against the six enemies.
func (g *Game) sub_e775() {
	R := &g.RAM
	R[3], R[4] = g.Rd(0x181a), g.Rd(0x181b)
	for x := byte(0); x != 6; x++ {
		if R[0x68+uint16(x)]&0x80 != 0 {
			continue
		}
		y := uint16(x) * 4
		a := g.Rd(0x1802+y) + 0x0c
		if a < R[3] || a-0x18 >= R[3] {
			continue
		}
		a = g.Rd(0x1803+y) + 0x0c
		if a < R[4] || a-0x18 >= R[4] {
			continue
		}
		R[0x68+uint16(x)] |= 0x10
		lvl := R[0x61+uint16(R[zPlayer])]
		if lvl >= 8 {
			lvl = 8
		}
		R[0x99+uint16(x)] = g.Rom(0xe7d7 + uint16(lvl))
		g.sub_eb61(0x0f)
	}
}

// sub_d899 runs the pepper spray animation (object 6 is the spray, $6e its timer).
func (g *Game) sub_d899() {
	R := &g.RAM
	if R[0x6e] == 0xff {
		return
	}
	g.sub_e775()
	R[0x6e]++
	y := uint16(R[0xba] >> 1)
	switch R[0x6e] {
	case 0x01:
		g.Wr(0x181c, g.Rom(0xd967+y))
		g.Wr(0x181d, g.Rom(0xd971+y))
		g.Wr(0x1818, g.Rom(0xd96c+y))
		g.Wr(0x1819, g.Rom(0xd985+y))
		x := uint16(R[zPlayer])
		g.carry = true
		R[0x2b+x] = g.bcdSbc(R[0x2b+x], 1)
		if R[zPlaying] != 0 {
			g.sub_cb6e()
		}
	case 0x09:
		g.sprayStep(y, 0xd971, 0xd976, 0xd98a)
	case 0x11:
		g.sprayStep(y, 0xd976, 0xd97b, 0xd98f)
	case 0x19:
		g.sprayStep(y, 0xd97b, 0xd980, 0xd994)
	case 0x22:
		R[0x6e] = 0xff
		g.Wr(0x1818, 0)
	}
}

// sprayStep changes the chef's sprite when it shows the expected pose, and sets the spray sprite.
func (g *Game) sprayStep(y, from, to, spray uint16) {
	if g.Rd(0x181d) == g.Rom(from+y) {
		g.Wr(0x181d, g.Rom(to+y))
	}
	g.Wr(0x1819, g.Rom(spray+y))
}

// sub_cb6e shows the number of pepper shakers left.
func (g *Game) sub_cb6e() {
	R := &g.RAM
	g.Wr(0x103a, 0x51)
	g.Wr(0x103b, 0x52)
	g.Wr(0x103c, 0x53)
	p := R[0x2b+uint16(R[zPlayer])]
	y := p >> 4
	if y != 0 {
		y++
	}
	g.Wr(0x105b, y)
	g.Wr(0x105c, p&0x0f+1)
}
