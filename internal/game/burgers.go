package game

// The burger ingredients: the per-frame state machine over the records at $0202
// (sub_e293), the helpers it uses, scoring (sub_ea8f) and the bonus items.
//
// A record is four bytes: type (low nibble) and state (high nibble), x (inverted),
// y and a spare byte ($0205+). The routines also look at the record that follows
// (offset +4), because the ingredients of one burger are listed top to bottom.
//
// States: 0 resting, 1 falling, 2..5 and 10..13 the bounce animation, 6 landing,
// 14 reached the plate, 15 done.

func (g *Game) stepRec(y byte) uint16 { return 0x202 + uint16(y) }

// sub_e293 runs the state machine over all ingredients.
func (g *Game) sub_e293() {
	R := &g.RAM
	R[0x200] = 0
	for {
		y := R[0x200]
		rec := R[0x202+uint16(y)]
		if rec == 0 {
			return
		}
		R[3] = rec & 0x0f
		R[4] = R[0x203+uint16(y)]
		R[5] = R[0x204+uint16(y)]
		y = g.recState(rec>>4, y)
		// e306: look for enemies that ride on this ingredient, then the next record
		g.sub_e6c1(y)
		R[0x200] += 4
	}
}

// recState handles one record; it returns the Y register the original leaves behind.
func (g *Game) recState(state, y byte) byte {
	R := &g.RAM
	r := uint16(y)
	switch state {
	case 0, 15:
	case 1:
		if g.sub_e619(y) != 0 {
			break
		}
		if g.sub_e671(y) != 0 {
			break
		}
		g.setPtr(5, g.drawBlockAuto(R[3], R[4], R[5]))
		R[0x204+r]++
		g.sub_ec8b()
	case 2:
		if R[0x205+r]&0x10 == 0 {
			g.carry = true
			R[5] = g.sbc(R[5], 8)
			R[0x204+r] = R[5]
			g.sub_e5cf(y, 0)
			R[0x205+r] |= 0x10
		} else if R[0x13]&3 == 0 {
			R[0x205+r] &= 0x0f
			R[0x202+r] = R[0x202+r]&0x0f | 0x30
		}
	case 3, 4, 5, 10, 11, 12, 13:
		// The original's entry points e5b5/e5be fall through into e5c7 (BEQ after a
		// non-zero load is never taken), so only frames 0 and 3 are ever used.
		frame := byte(3)
		if state == 10 {
			frame = 0
		}
		if R[0x205+r]&0x10 == 0 {
			g.sub_e5cf(y, frame)
			R[0x205+r] |= 0x10
		} else if R[0x13]&3 == 0 {
			R[0x205+r] &= 0x0f
			a := R[0x202+r]
			switch state {
			case 3:
				a = a&0x0f | 0x40
			case 4:
				a = (a | 0x50) & 0x5f
			case 5:
				a = (a | 0x60) & 0x6f
			case 10:
				a = a&0x0f | 0xb0
			case 11:
				a = (a | 0xc0) & 0xcf
			case 12:
				a = (a | 0xd0) & 0xdf
			case 13:
				a = (a | 0xe0) & 0xef
			}
			R[0x202+r] = a
		}
	case 6:
		g.setPtr(5, g.drawBlockAuto(R[3], R[4], R[5]))
		R[0x204+r]++
		if R[0x204+r]&7 != 0 {
			break
		}
		R[0x202+r] = R[0x202+r]&0x0f | 0x10
		R[0x205+r] &= 0x0f
		if R[0x206+r]&0x0f != 0x0f && R[0x206+r]&0xf0 != 0xf0 {
			break
		}
		R[0x205+r] = 0
		R[0x202+r] |= 0xf0
		g.sub_ea8f(0)
		g.sub_e6fd(y)
		y = g.sub_e8cb()
	case 14:
		g.setPtr(5, g.drawBlockAuto(R[3], R[4], R[5]))
		R[0x205+r] &= 0x0f
		if R[0x205+r] == 0 {
			R[0x202+r] &= 0x0f
			g.sub_ea8f(0)
			g.sub_e6fd(y)
			break
		}
		R[0x205+r]--
		if R[0x205+r] != 0 {
			R[0x202+r] = R[0x202+r]&0x0f | 0x10
			R[0x204+r] = g.adc(R[0x204+r], 2)
		} else {
			R[0x202+r] &= 0x0f
			g.sub_ea8f(0)
			g.sub_e6fd(y)
		}
	default:
		// states 7..9 are not used (the original executes BRK)
	}
	return y
}

// sub_e5cf redraws the ingredient with a shifted image: frame selects a
// y offset and a tile offset from the tables at $e60f / $e613.
func (g *Game) sub_e5cf(y, frame byte) {
	R := &g.RAM
	R[0x11] = frame
	R[5] = g.addc(R[5], g.Rom(0xe60f+uint16(frame)))
	p := g.drawBlock(R[3], R[4], R[5], false)
	p &^= 0x0400
	g.setPtr(5, p)
	add := g.Rom(0xe613 + uint16(frame))
	for _, o := range [...]uint16{1, 2, 0x21, 0x22} {
		g.Wr(p+o, g.Rd(p+o)+add)
	}
}

// sub_e619: a falling ingredient meets the one below it.
func (g *Game) sub_e619(y byte) byte {
	R := &g.RAM
	r := uint16(y)
	if R[5]&7 != 0 || R[0x207+r] != R[4] || R[0x208+r]-R[5] >= 9 {
		return 0
	}
	R[0x202+r] = R[0x202+r]&0x0f | 0x20
	if R[0x206+r]&0xf0 != 0xf0 && R[0x206+r]&0x0f != 0x0f {
		R[0x206+r] |= 0x10
		R[0x208+r] += 4
	}
	g.sub_eb61(0x0a)
	return 0xff
}

// sub_e671: a falling ingredient lands on a platform.
func (g *Game) sub_e671(y byte) byte {
	R := &g.RAM
	if R[5]&0x0f != 8 {
		return 0
	}
	R[0x12] = 0x10 - R[4]>>4
	idx := R[5]&0xf0 + R[0x12]
	cell, _ := g.mapCell(idx)
	if cell != 9 && cell != 8 {
		return 0
	}
	R[0x202+uint16(y)] = R[0x202+uint16(y)]&0x0f | 0xa0
	g.sub_eb61(0x0a)
	return 0xff
}

// sub_e6c1 moves enemies that stand on a falling ingredient along with it.
func (g *Game) sub_e6c1(y byte) {
	R := &g.RAM
	for x := byte(0); x != 6; x++ {
		if R[0x68+uint16(x)]&0xe0 != 0x40 {
			continue
		}
		R[0x0d] = R[0xb1+uint16(x)]
		if y != R[0x0d] {
			continue
		}
		R[0x0e] = y
		R[0x0f] = R[0x204+uint16(y)] - 0x0c
		g.Wr(0x1803+uint16(x)*4, R[0x0f])
	}
}

// sub_e6fd squashes the enemies riding on a landed ingredient and awards points.
func (g *Game) sub_e6fd(y byte) {
	R := &g.RAM
	R[0xc9] = 0
	for x := byte(0); x != 6; x++ {
		s := R[0x68+uint16(x)]
		if s&0x80 != 0 || s&0x40 == 0 {
			continue
		}
		R[0x0d] = R[0xb1+uint16(x)]
		if y != R[0x0d] {
			continue
		}
		R[0xc9]++
		g.emit(EvSquash, int(s&3), 0)
		R[0xca] = x
		R[0x68+uint16(x)] = s&0x0f | 0x20
		R[0x0e] = y
		kind := uint16(R[0x68+uint16(x)] & 3)
		g.Wr(0x1801+uint16(g.Rom(0xec6f+uint16(x))), g.Rom(0xe84f+kind))
		R[0x99+uint16(x)] = 0x7f
	}
	n := R[0xc9]
	if n == 0 {
		return
	}
	g.emit(EvDrop, 0, g.scorePoints(g.Rom(0xe766+uint16(n))))
	g.sub_ea8f(g.Rom(0xe766 + uint16(n)))
	sp := uint16(g.Rom(0xec6f + uint16(R[0xca])))
	g.Wr(0x1801+sp, g.Rom(0xe76c+uint16(n)))
	g.Wr(0x1800+sp, 1)
}

// sub_ec8b: a falling ingredient hits an enemy.
func (g *Game) sub_ec8b() {
	R := &g.RAM
	for x := 5; x >= 0; x-- {
		if R[0x68+x]&0x80 != 0 {
			continue
		}
		sp := uint16(g.Rom(0xec6f + uint16(x)))
		R[3], R[4] = g.Rd(0x1802+sp), g.Rd(0x1803+sp)
		y := uint16(R[0x200])
		a := g.sbc(R[0x203+y], 8)
		if !g.cmpGE(a, R[3]) {
			continue
		}
		a = g.sbc(a, 0x20)
		if g.cmpGE(a, R[3]) {
			continue
		}
		a = g.sbc(R[0x204+y], 5)
		if g.cmpGE(a, R[4]) {
			continue
		}
		a = g.adc(a, 0x0c)
		if !g.cmpGE(a, R[4]) {
			continue
		}
		g.emit(EvSquash, int(R[0x68+x]&3), 0)
		R[0x68+x] = R[0x68+x]&0x0f | 0x80
		R[0x99+x] = 0x42
		g.sub_eb61(0x0c)
	}
}

// sub_e8cb counts a finished burger piece; it brings in the bonus item or completes the level.
func (g *Game) sub_e8cb() (y byte) {
	R := &g.RAM
	p := uint16(R[zPlayer])
	g.emit(EvPiece, 0, 0)
	R[0x65+p]++
	y = R[0x63]
	R[5] = R[0x65+p]
	for _, base := range [...]uint16{0xe918, 0xe91e, 0xe924, 0xe92a} {
		if R[5] == g.Rom(base+uint16(y)) {
			g.sub_e1fe(R[zPlayer], y)
		}
	}
	if !g.cmpGE(R[5], g.Rom(0xe930+uint16(y))) {
		return y
	}
	R[0x6f] = R[0x6f]&0x0f | 0x80
	R[0xa0] = 0xff
	g.sub_eb61(5)
	return y
}

// sub_e1fe puts the bonus item on the screen.
func (g *Game) sub_e1fe(x, y byte) {
	R := &g.RAM
	R[0x11], R[0x12] = x, y
	R[0xc4] = g.Rom(0xe28b + uint16(R[0x64]))
	p := uint16(R[0x63]) * 2
	ptr := uint16(g.Rom(0xe279+p)) | uint16(g.Rom(0xe27a+p))<<8
	g.setPtr(3, ptr)
	t := g.Rom(0xe285 + uint16(R[0x63]))
	g.Wr(ptr, t)
	g.Wr(ptr+1, t+1)
	g.Wr(ptr+0x20, t+2)
	g.Wr(ptr+0x21, t+3)
	R[0x60] = 0
	g.sub_eb61(0x0b)
}

// sub_e9f5: the chef picks up the bonus item.
func (g *Game) sub_e9f5() {
	R := &g.RAM
	if R[0x60] != 0 {
		return
	}
	x := uint16(R[0x63])
	R[3], R[4] = g.Rom(0xea75+x), g.Rom(0xea7b+x)
	a := g.addc(g.Rd(0x181e), 8)
	if !g.cmpGE(a, R[3]) {
		return
	}
	if g.cmpGE(g.sbc(a, 0x10), R[3]) {
		return
	}
	a = g.addc(g.Rd(0x181f), 8)
	if !g.cmpGE(a, R[4]) {
		return
	}
	if g.cmpGE(g.sbc(a, 0x10), R[4]) {
		return
	}
	p := x * 2
	ptr := uint16(g.Rom(0xe279+p)) | uint16(g.Rom(0xe27a+p))<<8
	g.setPtr(3, ptr)
	k := uint16(g.Rom(0xea84 + x))
	R[5] = g.Rom(0xea81 + k)
	g.Wr(ptr, R[5])
	R[5]++
	g.Wr(ptr+1, R[5])
	g.Wr(ptr+0x20, 0)
	g.Wr(ptr+0x21, 0)
	R[0xc4] = 2
	R[0x60]++
	g.emit(EvBonus, 0, g.scorePoints(g.Rom(0xea8a+k)))
	g.sub_ea8f(g.Rom(0xea8a + k))
	pl := uint16(R[zPlayer])
	g.carry = false
	R[0x2b+pl] = g.bcdAdd(R[0x2b+pl], 1)
	g.sub_cb6e()
	g.sub_eb61(0x11)
}

// sub_e244 manages the bonus item's lifetime.
func (g *Game) sub_e244() {
	R := &g.RAM
	if R[0xc4] == 0 {
		return
	}
	g.sub_e9f5()
	if R[0x13]&0x3f != 0 {
		return
	}
	R[0xc4]--
	if R[0xc4] != 0 {
		return
	}
	p := uint16(R[0x63]) * 2
	ptr := uint16(g.Rom(0xe279+p)) | uint16(g.Rom(0xe27a+p))<<8
	g.setPtr(3, ptr)
	g.Wr(ptr, 0)
	g.Wr(ptr+1, 0)
	g.Wr(ptr+0x20, 0)
	g.Wr(ptr+0x21, 0)
	R[0x60] = 0
}

// sub_ea8f adds score entry idx (table $eb3b..) to the current player's score,
// updates the high score and awards extra chefs.
func (g *Game) sub_ea8f(idx byte) {
	R := &g.RAM
	R[4] = idx
	if R[zPlaying] == 0 {
		return
	}
	pl := R[zPlayer]
	y := uint16(g.Rom(0xeb5c + uint16(pl)))
	x := uint16(idx)
	g.carry = false
	R[0x2d+y] = g.bcdAdd(R[0x2d+y], g.Rom(0xeb3b+x))
	R[0x2e+y] = g.bcdAdd(R[0x2e+y], g.Rom(0xeb46+x))
	R[0x2f+y] = g.bcdAdd(R[0x2f+y], g.Rom(0xeb51+x))
	g.drawScore(pl)
	update := false
	switch {
	case R[0x35] < R[0x2f+y]:
		update = true
	case R[0x35] != R[0x2f+y]:
	case R[0x34] < R[0x2e+y]:
		update = true
	case R[0x34] != R[0x2e+y]:
	default:
		update = R[0x33] < R[0x2d+y]
	}
	if update {
		R[0x33], R[0x34], R[0x35] = R[0x2d+y], R[0x2e+y], R[0x2f+y]
		g.drawScore(2)
	}
	px := uint16(pl) * 2
	if R[0x2f+y] < R[0x5d+px] || R[0x2e+y] < R[0x5c+px] {
		return
	}
	g.carry = false
	R[0x5c+px] = g.bcdAdd(R[0x5c+px], R[0x5a])
	R[0x5d+px] = g.bcdAdd(R[0x5d+px], R[0x5b])
	R[0x29+uint16(pl)]++
	g.sub_cb2a()
	g.sub_eb61(2)
}


// sub_ebad runs the squashed enemies' animation and removes them at the end.
func (g *Game) sub_ebad() {
	R := &g.RAM
	for x := 5; x >= 0; x-- {
		if R[0x68+x]&0xf0 != 0x80 {
			continue
		}
		sp := uint16(g.Rom(0xec6f + uint16(x)))
		if R[0x99+x] == 1 {
			kind := uint16(R[0x68+x] & 3)
			R[0xbe+kind]--
			R[0x03] = byte(x)
			R[0x68+x] = 0xff
			g.Wr(0x1800+sp, 0)
			continue
		}
		R[0x99+x]--
		kind := uint16(R[0x68+x] & 3)
		switch R[0x99+x] {
		case 0x40:
			g.Wr(0x1801+sp, g.Rom(0xec76+kind))
		case 0x38:
			g.Wr(0x1801+sp, g.Rom(0xec79+kind))
		case 0x30:
			g.Wr(0x1801+sp, g.Rom(0xec7c+kind))
		case 0x28:
			g.Wr(0x1801+sp, g.Rom(0xec7f+kind))
		case 0x20:
			g.emit(EvDrop, 0, g.scorePoints(g.Rom(0xec85+kind)))
			g.sub_ea8f(g.Rom(0xec85 + kind))
			g.Wr(0x1801+sp, g.Rom(0xec82+kind))
			g.Wr(0x1800+sp, 1)
		}
	}
}

// sub_e7e2 times the stunned enemies (blink, then recover).
func (g *Game) sub_e7e2() {
	R := &g.RAM
	for x := byte(0); x != 6; x++ {
		s := R[0x68+uint16(x)]
		if s&0x80 != 0 || s&0x10 == 0 {
			continue
		}
		var tile byte
		if R[0x99+uint16(x)] == 1 {
			R[0x68+uint16(x)] = s & 0xef
			tile = g.Rom(0xe85d + uint16(R[0x68+uint16(x)]&3))
		} else {
			R[0x99+uint16(x)]--
			if R[0x99+uint16(x)]&0x0f != 0 {
				continue
			}
			if R[0x99+uint16(x)]&0x10 != 0 {
				tile = g.Rom(0xe84f + uint16(s&3))
			} else {
				tile = g.Rom(0xe856 + uint16(s&3))
			}
		}
		R[3] = tile
		g.Wr(0x1801+uint16(x)*4, tile)
	}
}

// sub_e863: an enemy touches the chef.
func (g *Game) sub_e863() {
	R := &g.RAM
	if R[0x6f]&0x80 != 0 {
		return
	}
	R[3], R[4] = g.Rd(0x181e), g.Rd(0x181f)
	for x := byte(0); x != 6; x++ {
		if R[0x68+uint16(x)]&0xf0 != 0 {
			continue
		}
		y := uint16(x) * 4
		a := g.addc(g.Rd(0x1802+y), 8)
		if !g.cmpGE(a, R[3]) || g.cmpGE(g.sbc(a, 0x10), R[3]) {
			continue
		}
		a = g.addc(g.Rd(0x1803+y), 4)
		if !g.cmpGE(a, R[4]) || g.cmpGE(g.sbc(a, 0x0d), R[4]) {
			continue
		}
		g.sub_eb61(0x10)
		R[0x6f] |= 0xf0
		R[0xa0] = 0xff
		g.sub_eb61(0)
	}
}

// sub_e938: the chef is dying; blink the sprite and count the pause.
func (g *Game) sub_e938() {
	R := &g.RAM
	if R[0x6f]&0xf0 != 0x80 {
		return
	}
	if R[0xa0] == 0 {
		R[zRestart]++
		return
	}
	R[0xa0]--
	if R[0xa0]&0x0f != 0 {
		return
	}
	if R[0xa0]&0x10 != 0 {
		g.Wr(0x181d, 0x47)
	} else {
		g.Wr(0x181d, 0x4c)
	}
}

// sub_e96f: the chef has been caught; play the death animation.
func (g *Game) sub_e96f() {
	R := &g.RAM
	if R[zEndGame]&0x80 != 0 {
		return
	}
	g.cmp(R[0x6f]&0xf0, 0xf0)
	if R[0x6f]&0xf0 != 0xf0 {
		return
	}
	x := 0x1b
	for ; x >= 0; x-- {
		if g.cmp(R[0xa0], g.Rom(0xe9bb+uint16(x))) {
			break
		}
	}
	if x < 0 {
		R[0xa0]--
		return
	}
	g.Wr(0x181d, g.Rom(0xe9d7+uint16(x)))
	R[0xa0]--
	if x == 0 {
		R[0x29+uint16(R[zPlayer])]--
		R[zEndGame] = 0xff
		return
	}
	g.carry = x >= 0x18
	if x == 0x18 {
		g.sub_eb61(0x10)
	}
}
