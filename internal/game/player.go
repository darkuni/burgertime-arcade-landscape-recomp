package game

// The chef (object 7): joystick and pepper button (sub_d27e, sub_d2b0, sub_d339).

// sub_d27e reads the controls and moves the chef.
func (g *Game) sub_d27e() {
	R := &g.RAM
	if R[0x6f]&0x80 != 0 {
		return
	}
	if R[zPlaying] == 0 {
		g.sub_d339()
		return
	}
	x := R[zPlayer]
	y := uint16(R[zPlayer])
	if g.Rd(0x4003)&0x40 == 0 {
		y = 0
	}
	pad := func() byte { return ^g.Rd(0x4000 + y) }
	if pad()&InButton != 0 {
		if R[0xb9] == 0 {
			p := R[zPepper+uint16(x)]
			if p != 0 {
				R[0xb9] = p
				g.sub_d2b0()
				return
			}
			g.sub_eb61(0x0e)
		}
	} else {
		R[0xb9] = 0
	}
	d := g.Rom(0xd35a + uint16(pad()&0x0f))
	if g.cmp(d, R[0xbb]) {
		// same direction as before
	} else {
		R[0xb0] = g.asl(g.asl(g.asl(g.asl(d))))
	}
	R[0xbb] = d
	if d != 0 {
		R[0xba] = d
	}
	if R[0xbb] != 0 {
		r := g.sub_d4c6(7)
		if !r.z {
			return
		}
	}
	g.sub_d36c(7)
	a := R[0xb0] & 0xf0
	if a == 0 {
		return
	}
	snd := byte(6)
	if g.cmp(a, 0x60); !g.carry {
		snd = 7
	}
	g.sub_eb61(snd)
}

// sub_d2b0 throws pepper in the direction the chef faces.
func (g *Game) sub_d2b0() {
	R := &g.RAM
	R[0x6e] = 0
	R[0xa0] = 5
	d := uint16(R[0xba])
	g.Wr(0x181a, g.addc(g.Rom(0xd350+d), g.Rd(0x181e)))
	g.Wr(0x181b, g.addc(g.Rom(0xd351+d), g.Rd(0x181f)))
	g.sub_eb61(0x0d)
}

// sub_d339 moves the chef in the demonstration (no game running).
func (g *Game) sub_d339() {
	R := &g.RAM
	if r := g.sub_d4c6(7); !r.z {
		g.Wr(0x181d, 0x47)
		R[0xc6]++
	}
	g.sub_d36c(7)
}
