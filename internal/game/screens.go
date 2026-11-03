package game

// Title / attract screens.

// sub_c7dd reads the difficulty switches into the game settings.
func (g *Game) sub_c7dd() {
	R := &g.RAM
	y := ^g.Rd(0x4004)
	x := byte(2)
	if y&1 != 0 {
		x = 4
	}
	R[0x29], R[0x2a] = x, x
	g.carry = y&1 != 0
	x = (y >> 1) & 3
	R[0x5a], R[0x5c], R[0x5e] = g.Rom(0xc80e+uint16(x)), g.Rom(0xc80e+uint16(x)), g.Rom(0xc80e+uint16(x))
	R[0x5b], R[0x5d], R[0x5f] = g.Rom(0xc812+uint16(x)), g.Rom(0xc812+uint16(x)), g.Rom(0xc812+uint16(x))
	g.sub_ece3()
}

// sub_c40f is the bonus life information screen.
func (g *Game) sub_c40f() {
	R := &g.RAM
	g.sub_c976()
	g.sub_ccbd()
	g.printText(0xcee9)
	g.printText(0xcf37)
	g.printText(0xcfcc)
	g.Wr(0x12d5, R[0x5b]+1)
	g.Wr(0x12d6, R[0x5a]>>4+1)
	g.Wr(0x12d7, 1)
	g.Wr(0x12d8, 1)
	g.Wr(0x12d9, 1)
}

// sub_c44c is the high score table screen.
func (g *Game) sub_c44c() {
	R := &g.RAM
	g.sub_c976()
	g.sub_ccbd()
	g.printText(0xcee9)
	g.printText(0xcffd)
	p := uint16(0x11c9)
	for x := uint16(0x12); x < 0x21; x += 3 {
		g.Wr(p, R[0x36+x])
		g.Wr(p+1, R[0x37+x])
		g.Wr(p+2, R[0x38+x])
		p += 0x40
	}
	g.carry = true
	for x := byte(3); x <= 7; x++ {
		g.drawScore(x)
	}
}

// sub_c4ae is the screen that introduces the characters.
func (g *Game) sub_c4ae() {
	g.sub_c976()
	g.sub_ccbd()
	g.printText(0xcee9)
	g.printText(0xc5e5)
	g.sub_c51e(1)
	// tile codes of the big pictures: 2x2 blocks of consecutive codes
	pics := []struct {
		addr uint16
		code byte
	}{{0x1188, 0x10}, {0x1208, 0x6c}, {0x1268, 0x90}, {0x12c8, 0xc4}}
	for _, p := range pics {
		g.Wr(p.addr, p.code)
		g.Wr(p.addr+1, p.code+1)
		g.Wr(p.addr+0x20, p.code+2)
		g.Wr(p.addr+0x21, p.code+3)
	}
	g.waitFrames(0xff)
	g.waitFrames(0xff)
	g.sub_c51e(0)
}
