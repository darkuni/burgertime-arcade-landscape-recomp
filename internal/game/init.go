package game

// sub_c362: power-on initialisation: clears video RAM, the zero page and the two
// object table pages, loads the default high score table and the palette.
func (g *Game) sub_c362() {
	g.VRAM = [0x400]byte{}
	g.CRAM = [0x400]byte{}
	for i := 0x01; i <= 0xe1; i++ {
		g.RAM[i] = 0
	}
	for i := 0; i < 0x100; i++ {
		g.RAM[0x200+i] = 0
		g.RAM[0x300+i] = 0
	}
	g.RAM[0x35] = 0x02
	g.RAM[0x34] = 0x80
	g.RAM[0x33] = 0x00
	// default high score table: 36 bytes of ROM data to $36..$59
	for i := 0; i < 0x24; i++ {
		g.RAM[0x36+i] = g.Rom(0xc3e9 + uint16(i))
	}
	for i := 0; i < 16; i++ {
		g.Palette[i] = g.Rom(0xc3d9 + uint16(i))
	}
	g.RAM[0xf8] = 0
	g.RAM[0xf9] = 0
}

// sub_c34d: resets the sound board and a few flags.
func (g *Game) sub_c34d() {
	g.RAM[0x25] = 0
	g.Wr(0x4003, 0) // sound command 0: silence
	g.RAM[0x27] = 0x80
}

// sub_c333: swaps the two player state pages ($0200 and $0300).
func (g *Game) sub_c333() {
	for i := 0; i < 0x100; i++ {
		g.RAM[0x200+i], g.RAM[0x300+i] = g.RAM[0x300+i], g.RAM[0x200+i]
	}
}

// sub_c976: blanks the characters and colours (and so the sprites), then waits a frame.
func (g *Game) sub_c976() {
	g.VRAM = [0x400]byte{}
	g.CRAM = [0x400]byte{}
	g.waitFrames(1)
}

// sub_ccbd: clears the top nibbles of work RAM pages 4-7, turns the background layer off and waits a frame.
func (g *Game) sub_ccbd() {
	for i := 0x400; i < 0x800; i++ {
		g.RAM[i] &= 0x0f
	}
	g.Wr(0x4004, 0)
	g.waitFrames(1)
}

// sub_c51e: stores A in 16 colour RAM cells (the cells under the logo's burger).
func (g *Game) sub_c51e(a byte) {
	for _, ad := range [...]uint16{
		0x1588, 0x1589, 0x15a8, 0x15a9, 0x1608, 0x1609, 0x1628, 0x1629,
		0x1668, 0x1669, 0x1688, 0x1689, 0x16c8, 0x16c9, 0x16e8, 0x16e9,
	} {
		g.Wr(ad, a)
	}
}

// printText draws the text records at ptr (sub_ca8f). A record is a screen
// address followed by character codes; $fd starts the next row, $fe the next
// record and $ff ends the text. While $c8 is set every character is followed by a pause.
func (g *Game) printText(ptr uint16) {
	for {
		dest := uint16(g.Rom(ptr)) | uint16(g.Rom(ptr+1))<<8
		off := uint16(0)
		y := uint16(2)
		for {
			c := g.Rom(ptr + y)
			y++
			switch c {
			case 0xff:
				g.RAM[0xc8] = 0
				g.carry = true // CMP #$ff
				return
			case 0xfe:
				ptr += y
				goto next
			case 0xfd:
				dest += 0x20
				off = 0
			default:
				g.Wr(dest+off, c)
				off++
				if g.RAM[0xc8] != 0 {
					g.waitFrames(10)
				}
			}
		}
	next:
	}
}

// drawScore shows score slot n: 0 = player 1, 1 = player 2, 2 = high score,
// 3.. = the high score table rows (sub_ca21).
func (g *Game) drawScore(n byte) {
	g.drawScoreAt(uint16(n)*2, n*3)
}

// drawScoreAt is sub_ca2d: y selects the screen address word in the table at $ca73, x the first score byte ($2d+x).
func (g *Game) drawScoreAt(y uint16, x byte) {
	ptr := uint16(g.Rom(0xca73+y)) | uint16(g.Rom(0xca74+y))<<8
	pos := int8(5)
	for pos >= 0 {
		b := g.RAM[0x2d+int(x)]
		g.Wr(ptr+uint16(pos), (b&0x0f)+1)
		pos--
		g.Wr(ptr+uint16(pos), b>>4+1)
		x++
		pos--
	}
	// leading zeros become blanks
	for i := uint16(0); i < 5; i++ {
		if g.Rd(ptr+i) != 1 {
			break
		}
		g.Wr(ptr+i, 0)
	}
}

// sub_cce9: loads the object layout of the current level of the current player into $0202...
// The level number (1-based, in $61/$62) repeats every six levels.
func (g *Game) sub_cce9() {
	lvl := int(g.RAM[0x61+g.RAM[zPlayer]]) - 1
	lvl &= 0xff
	for lvl >= 6 {
		lvl -= 6
	}
	p := uint16(g.Rom(0xcebd+uint16(lvl)*2)) | uint16(g.Rom(0xcebe+uint16(lvl)*2))<<8
	// records of three bytes at a stride of four; the fourth byte is left alone
	x := byte(0)
	for y := uint16(0); ; y++ {
		v := g.Rom(p + y)
		if v == 0xff {
			break
		}
		g.RAM[0x202+int(x)] = v
		x++
		if x&3 == 3 {
			x++
			g.RAM[0x202+int(x)] = 0
		}
	}
	for {
		g.RAM[0x202+int(x)] = 0
		x++
		if x == 0 {
			break
		}
	}
}
