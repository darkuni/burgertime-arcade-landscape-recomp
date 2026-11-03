package game

// Main program entry points (labels of the original program).
const (
	entryAttract   = 0xc02f // attract mode / title sequence
	entryGameStart = 0xc06a // a game is being started
	entryLevel     = 0xc077 // set up the current level
	entryMainLoop  = 0xc081 // the per-frame game loop
)

// Zero page variables used by the main program.
const (
	zBooted   = 0x01 // set once the program has initialised itself (enables the coin interrupt)
	zFrame    = 0x13 // frame counter (low byte)
	zFrame16L = 0x14 // 16 bit frame counter
	zFrame16H = 0x15
	zRandom   = 0x16 // random seed, stirred with the inputs every frame
	zCredWait = 0x1a // non zero while credits are waiting and the start buttons are live
	zPlaying  = 0x1b // non zero while a game is in progress
	zInput    = 0x1c // set once the first level is running (enables ... )
	zCredits  = 0x1d // credits, BCD
	zPlayer   = 0x1f // current player (0 or 1)
	zNewLife  = 0x20 // set when a player starts a new life
	zTwoPlay  = 0x21 // non zero for a two player game
	zLives    = 0x29 // lives of player 1; player 2 at +1 (bit 7 set = out of lives)
	zPepper   = 0x2b // pepper shakers of player 1; player 2 at +1
	zRestart  = 0xc5 // set when the level has to be restarted
	zEndGame  = 0xc6 // set when the current player's turn is over
)

// reset is the main program, starting at the reset vector (sub_c00d).
func (g *Game) reset() {
	g.RAM[zBooted] = 0
	g.RAM[0xfa] = 0
	g.sub_c362()
	g.sub_c34d()
	g.RAM[zBooted] = 1
	g.RAM[zCredWait] = 0
	g.sub_c7dd()
	entry := uint16(entryAttract)
	for {
		entry = g.runFrom(entry)
	}
}

// runFrom runs the main program from one of its entry points until the program
// performs a non-local jump, whose destination is returned.
func (g *Game) runFrom(entry uint16) (next uint16) {
	defer func() {
		if r := recover(); r != nil {
			if u, ok := r.(unwind); ok {
				next = u.to
				return
			}
			panic(r)
		}
	}()
	st := entry
	for {
		switch st {
		case entryAttract:
			st = g.attract()
		case entryGameStart:
			g.sub_c7dd()
			g.sub_c818()
			g.RAM[zInput] = 1
			st = entryLevel
		case entryLevel:
			if g.RAM[zPlaying] == 0 {
				st = entryAttract
				continue
			}
			g.sub_c837()
			st = entryMainLoop
		case entryMainLoop:
			st = g.gameFrame()
		case 0xc15a:
			st = g.turnOver()
		case 0xc221:
			st = g.gameOver()
		default:
			panic("bad entry")
		}
	}
}

// attract runs the title / demo sequence (c02f..c069) and falls into the game start.
func (g *Game) attract() uint16 {
	g.Wr(0x5005, 0xfe)
	g.VidCtl = 0
	g.RAM[zPlaying] = 0
	g.RAM[zNewLife] = 0
	g.RAM[zInput] = 0
	g.sub_c44c()
	g.waitFrames(0xff)
	g.waitFrames(0x3f)
	g.sub_c40f()
	g.waitFrames(0xff)
	g.sub_c4ae()
	g.sub_c976()
	g.sub_c552()
	g.waitFrames(0x3f)
	g.sub_c62c()
	return entryGameStart
}

// vblankWork is the work the program does at the start of every vertical blank:
// the coin interrupt window, coin credit handling, pause handling and start button handling.
// It returns the processor status left by the last routine (the program's random
// number generator consumes the carry flag).
func (g *Game) vblankWork() {
	g.vblankStart()
	g.sub_d148()
	g.sub_ecf9()
}

// vblankStart blocks until the start of vertical blank and then opens the
// interrupt window in which a pending coin interrupt is taken.
func (g *Game) vblankStart() {
	g.waitVblank()
	g.Frames++
	if g.coinIRQ {
		g.coinIRQ = false
		g.sub_d046()
	}
}

// sub_ed71: wait X frames, doing only the coin credit work.
func (g *Game) pauseWait(n byte) {
	for {
		g.vblankStart()
		g.sub_d148()
		n--
		if n == 0 {
			return
		}
	}
}

// waitFrames waits n video frames (0 means 256) doing the per-frame work (sub_caff).
func (g *Game) waitFrames(n byte) {
	for {
		g.vblankWork()
		if g.RAM[zCredWait] != 0 {
			g.sub_d171()
		}
		n--
		if n == 0 {
			return
		}
	}
}

// gameFrame is one iteration of the in-game loop (c081..c154) and returns the next state.
func (g *Game) gameFrame() uint16 {
	g.vblankWork()
	g.sub_d171()
	g.RAM[zFrame]++
	if g.RAM[zFrame]&0x3f == 0 {
		// (the original has a padding store here)
	}
	g.RAM[zFrame16L]++
	if g.RAM[zFrame16L] == 0 {
		g.RAM[zFrame16H]++
	}
	// random number seed: seed = seed + 1 + P1 + P2 + carry
	g.RAM[zRandom]++
	sum := int(g.P1) + int(g.P2) + int(g.Carry())
	carry := sum > 0xff
	sum2 := int(byte(sum)) + int(g.RAM[zRandom])
	if carry {
		sum2++
	}
	g.RAM[zRandom] = byte(sum2)

	g.sub_c9b9()
	g.sub_d27e()
	g.sub_d899()
	g.sub_d99b()
	if g.RAM[zPlaying] != 0 {
		g.sub_dc2b()
		g.sub_dcbe()
		g.sub_dd7a()
	}
	g.sub_de68()
	g.sub_ebad()
	g.sub_e7e2()
	g.sub_e244()
	g.sub_e293()
	g.sub_e938()
	g.sub_e863()
	g.sub_e96f()

	p := g.RAM[zPlayer]
	if g.RAM[zPepper+p] != 0 {
		if g.RAM[0xf8+p] != 0 {
			g.RAM[0xf8+p] = 0
			g.sub_eb61(0x19)
		}
	} else {
		if g.RAM[0xf8+p] == 0 {
			g.RAM[0xf8+p] = 0x18
			g.sub_eb61(0x18)
		}
	}
	p = g.RAM[zPlayer]
	if g.RAM[zPepper+p] == 0 {
		a := g.RAM[zFrame] & 0x3f
		if a == 0 {
			g.printText(0xc12c)
		} else {
			g.carry = a >= 0x0f // CMP #$0f
			if a == 0x0f {
				g.printText(0xc132)
			}
		}
	}
	if g.RAM[zEndGame] != 0 {
		return 0xc15a
	}
	if g.RAM[zRestart] != 0 {
		return entryLevel
	}
	return entryMainLoop
}

// turnOver handles the end of a player's turn (c15a): next player, same player's next life, or game over.
func (g *Game) turnOver() uint16 {
	g.RAM[zEndGame] = 0
	if g.RAM[zPlaying] == 0 {
		g.waitFrames(0x3f)
		return entryAttract
	}
	g.sub_c976()
	g.sub_ccbd()
	l1, l2 := g.RAM[zLives], g.RAM[zLives+1]
	if g.RAM[zTwoPlay] == 0 {
		if l1&0x80 != 0 {
			return 0xc221
		}
		g.RAM[zNewLife] = 1
		g.sub_c282()
		return entryLevel
	}
	if g.RAM[zPlayer] == 0 {
		if l1&l2&0x80 != 0 {
			return 0xc221
		}
		if l2&0x80 != 0 { // player 2 is out: player 1 carries on
			g.sub_c29d()
			g.RAM[zNewLife] = 1
			return entryLevel
		}
		if l1&0x80 != 0 {
			g.sub_c2c5()
			g.sub_c976()
		}
		g.sub_c333()
		g.RAM[zPlayer] = 1
		g.RAM[zNewLife] = 1
		if g.DSW1&0x40 != 0 { // cocktail cabinet: turn the picture round for player 2
			g.Wr(0x5005, 0)
			g.VidCtl = 1
		}
		g.sub_c2b1()
		return entryLevel
	}
	if l1&l2&0x80 != 0 {
		return 0xc221
	}
	if l1&0x80 != 0 { // player 1 is out: player 2 carries on
		g.RAM[zNewLife] = 1
		g.sub_c2b1()
		return entryLevel
	}
	if l2&0x80 != 0 {
		g.sub_c2dd()
		g.sub_c976()
	}
	g.sub_c333()
	g.RAM[zPlayer] = 0
	g.RAM[zNewLife] = 1
	g.Wr(0x5005, 0xfe)
	g.VidCtl = 0
	g.sub_c29d()
	return entryLevel
}

// gameOver shows the game over messages and enters the high scores (c221).
func (g *Game) gameOver() uint16 {
	g.sub_c2f5()
	g.sub_c976()
	g.Wr(0x5005, 0xfe)
	g.VidCtl = 0
	g.RAM[0xcb] = 0
	g.RAM[0xcc], g.RAM[0xcd], g.RAM[0xce] = g.RAM[0x2d], g.RAM[0x2e], g.RAM[0x2f]
	g.sub_f1af()
	g.sub_c976()
	if g.DSW1&0x40 != 0 {
		g.Wr(0x5005, 0)
		g.VidCtl = 1
	}
	g.RAM[0xcb] = 1
	g.RAM[0xcc], g.RAM[0xcd], g.RAM[0xce] = g.RAM[0x30], g.RAM[0x31], g.RAM[0x32]
	g.sub_f1af()
	g.RAM[zPlaying] = 0
	if g.RAM[zCredits] != 0 {
		g.RAM[zCredWait]++
	}
	return entryAttract
}
