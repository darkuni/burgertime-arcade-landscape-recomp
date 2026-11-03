package game

// Credits, coins, sound commands and the start buttons.

// bcdAdd is SED; ADC (valid BCD operands).
func (g *Game) bcdAdd(a, b byte) byte {
	lo := int(a&0x0f) + int(b&0x0f)
	if g.carry {
		lo++
	}
	if lo > 9 {
		lo += 6
	}
	r := int(a&0xf0) + int(b&0xf0) + lo
	g.carry = false
	if r > 0x99 {
		r += 0x60
		g.carry = true
	}
	return byte(r)
}

// bcdSbc is SED; SBC (valid BCD operands, carry set = no borrow).
func (g *Game) bcdSbc(a, b byte) byte {
	lo := int(a&0x0f) - int(b&0x0f)
	if !g.carry {
		lo--
	}
	hi := int(a>>4) - int(b>>4)
	if lo < 0 {
		lo -= 6
		hi--
	}
	if hi < 0 {
		hi -= 6
		g.carry = false
	} else {
		g.carry = true
	}
	return byte(hi<<4) | byte(lo&0x0f)
}

// sub_eb61 sends sound number id to the sound board (only while a game is being played).
func (g *Game) sub_eb61(id byte) (a byte) {
	R := &g.RAM
	if R[zPlaying] == 0 {
		return id
	}
	v := g.Rom(0xeb91 + uint16(id))
	g.cmp(v, 4)
	if v == 4 {
		// the pepper sound: the shaker sound depends on whether the player has pepper
		if R[0xf8+uint16(R[zPlayer])] != 0 {
			v = 5
		} else {
			v = 3
		}
	}
	g.Wr(0x4003, v)
	return v
}

// sub_faa0: clear the coin counter and send sound id.
func (g *Game) sub_faa0(id byte) byte {
	g.RAM[0x1e] = id
	return g.sub_eb61(id)
}

// sub_d148 is called every frame: once a coin switch has closed and reopened
// the coin is credited.
func (g *Game) sub_d148() {
	if g.RAM[0xfa] == 0 {
		return
	}
	if g.Rd(0x4002)&0xc0 != 0 {
		return
	}
	g.RAM[0xfa] = 0
	g.coinCredit()
}

// coinCredit is the coin handler at $fa7e / $d0a1: it counts the coin, applies
// the coinage settings and adds a credit.
func (g *Game) coinCredit() {
	R := &g.RAM
	R[0x02] = ^g.Rd(0x4004) & 0xe0
	y := uint16(R[0x02] >> 5)
	R[0x1e]++
	g.Wr(0x4003, 0x1b) // coin sound
	var x byte
	switch {
	case R[0x02] == 0x80: // free play: the coin switch bits select the amount
		x = 5
		switch R[0x26] {
		case 0x80:
		case 0x40:
			x = 6
		case 0xc0:
			x = 7
		default:
			return
		}
	default:
		a := ^g.Rd(0x4003) & 0x0f
		coin2 := R[0x26]&0x80 != 0
		R[0x26] <<= 1
		if coin2 {
			a >>= 2
		}
		a &= 3
		switch a {
		case 0:
			x = 0
		case 1:
			x = 1
		case 2:
			x = 2
			if R[0x02] == 0x60 {
				x = 4
			}
		default:
			x = 3
			g.cmp(R[0x1e], 2)
			if R[0x1e] < 2 {
				return
			}
			R[0x1e]--
		}
	}
	g.carry = false
	v := g.bcdAdd(R[0x1d], g.Rom(0xd15f+uint16(x)))
	if m := g.Rom(0xd167 + y); !g.cmpGE(v, m) {
		R[0x1d] = v
	} else {
		R[0x1d] = m
	}
	R[0x1e]--
	if R[zPlaying] != 0 {
		return
	}
	R[zCredWait] = 1
	g.jumpTo(entryAttract)
}

// sub_ece3: the number of pepper shakers per player at the start of a game.
func (g *Game) sub_ece3() {
	R := &g.RAM
	R[0x2b], R[0x2c] = 5, 5
	if g.Rd(0x4004)&0x10 == 0 {
		R[0x2b]--
	}
}

// sub_ecf9 handles the pause / quit switch during a game.
func (g *Game) sub_ecf9() {
	R := &g.RAM
	if R[zPlaying] == 0 {
		return
	}
	if ^g.Rd(0x4002)&0x04 == 0 {
		return
	}
	g.sub_faa0(0)
	g.sub_c976()
	g.sub_ccbd()
	g.printText(0xed8c)
	if R[zCredits] != 0 {
		g.sub_ed63()
		g.carry = true
		R[zCredits] = g.bcdSbc(R[zCredits], 1)
		if R[zCredits] == 0 {
			g.leaveGame(0)
		}
		g.leaveGame(1)
	}
	g.sub_ed63()
	switch R[zCredits] {
	case 0:
		g.leaveGame(0)
	case 1:
	default:
		g.carry = true
		R[zCredits] = g.bcdSbc(R[zCredits], 1)
	}
	g.leaveGame(1)
}

// leaveGame returns to the attract mode with the credit-waiting flag set to v.
func (g *Game) leaveGame(v byte) {
	g.RAM[zCredWait] = v
	g.jumpTo(entryAttract)
}

func (g *Game) sub_ed63() {
	g.pauseWait(0xbd)
	g.pauseWait(0x7e)
}

// sub_d271: take one credit away.
func (g *Game) sub_d271() {
	g.carry = true
	g.RAM[zCredits] = g.bcdSbc(g.RAM[zCredits], 1)
}

// sub_d242 shows the credit count.
func (g *Game) sub_d242() {
	R := &g.RAM
	g.lsr4(R[zCredits])
	g.Wr(0x13b3, R[zCredits]&0x0f+1)
}

// sub_d214 draws the "CREDIT" text if it is not on the screen yet and then the count.
func (g *Game) sub_d214() {
	if g.cmp(g.Rd(0x1367), g.Rom(0xd258)) {
		return
	}
	for i := uint16(0); i < 0x11; i++ {
		g.Wr(0x1367+i, g.Rom(0xd258+i))
	}
	for i := uint16(0); i < 6; i++ {
		g.Wr(0x13ab+i, g.Rom(0xd269+i))
	}
	g.carry = true
	g.sub_d242()
}

// sub_d171 reads the start buttons while credits are waiting.
func (g *Game) sub_d171() {
	R := &g.RAM
	if R[zCredWait] == 0 {
		return
	}
	if R[zInput] == 0 {
		g.sub_d214()
		g.sub_d242()
	}
	if g.Rd(0x4002)&1 == 0 {
		g.sub_d271()
		g.startGame(false)
	}
	if g.Rd(0x4002)&2 != 0 {
		return
	}
	if g.cmpGE(R[zCredits], 2) {
		g.sub_d271()
		g.sub_d271()
		g.startGame(true)
	}
}

// startGame begins a one or two player game (d198 / d1cc).
func (g *Game) startGame(two bool) {
	R := &g.RAM
	if two {
		R[zTwoPlay] = 1
	} else {
		R[zTwoPlay] = 0
	}
	R[zNewLife] = 0
	R[zInput] = 0
	R[zPlaying] = 1
	R[zCredWait] = 0
	g.sub_c976()
	g.sub_ccbd()
	if two {
		g.printText(0xc302)
	}
	g.printText(0xc318)
	g.waitFrames(0x1f)
	for i := 0; i < 6; i++ {
		R[0x2d+i] = 0
	}
	g.jumpTo(entryGameStart)
}
