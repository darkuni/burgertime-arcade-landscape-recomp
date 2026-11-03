package game

// Relaxed controls (an option; the default is the original behaviour).
//
// The arcade joystick is strictly four-way: two directions at once do nothing at
// all, and the chef only climbs when he stands within a few pixels of a ladder
// column (and only walks when he is level with a floor). Relaxed controls keep
// the game's own rules but forgive near misses:
//
//   - two directions at once (a rolled thumb, a stick pushed diagonally) pick
//     whichever one the chef can use, preferring up/down;
//   - a direction the chef cannot take yet is replaced by a small step along the
//     other axis when that lines him up (walk the last few pixels to the ladder,
//     or climb the last few pixels to the floor), and the direction itself is
//     taken as soon as he is lined up;
//   - opposite directions cancel.
//
// Everything is decided by asking the original movement test (sub_d4c6) on a
// scratch copy of the machine, so the maze rules stay exactly the arcade's.

const relaxedSnap = 8 // pixels of forgiveness along the other axis

// chef coordinates in the sprite RAM window: $181e is the walking axis (Right
// decreases it), $181f the climbing axis (Up decreases it).
const (
	chefWalk  = 0x181e
	chefClimb = 0x181f
)

// direction codes as the original stores them (d4eb table).
const (
	dirRight = 2
	dirLeft  = 4
	dirUp    = 6
	dirDown  = 8
)

func dirBits(d byte) byte {
	switch d {
	case dirRight:
		return InRight
	case dirLeft:
		return InLeft
	case dirUp:
		return InUp
	}
	return InDown
}

// canMove reports whether the original maze test lets the chef start moving in
// direction d from his position shifted by dw / dc along the walking / climbing axes.
func (g *Game) canMove(d byte, dw, dc int) bool {
	c := *g // scratch copy: the test writes working variables
	c.RAM[0xb0] = d<<4 | c.RAM[0xb0]&0x0f
	if dw != 0 {
		c.Wr(chefWalk, byte(int(c.Rd(chefWalk))+dw))
	}
	if dc != 0 {
		c.Wr(chefClimb, byte(int(c.Rd(chefClimb))+dc))
	}
	return c.sub_d4c6(7).z
}

// usable returns the direction to send to the original for wish d: d itself, a
// step that lines the chef up for d, or 0 when neither works.
func (g *Game) usable(d byte) byte {
	if g.canMove(d, 0, 0) {
		return d
	}
	for k := 1; k <= relaxedSnap; k++ {
		for _, s := range []int{-1, 1} {
			var ok bool
			var step byte
			if d == dirRight || d == dirLeft { // needs to be level with the floor: shift on the climbing axis
				ok = g.canMove(d, 0, s*k)
				step = dirUp
				if s > 0 {
					step = dirDown
				}
			} else { // needs to be on the ladder column: shift on the walking axis
				ok = g.canMove(d, s*k, 0)
				step = dirRight
				if s > 0 {
					step = dirLeft
				}
			}
			if ok && g.canMove(step, 0, 0) {
				return step
			}
		}
	}
	return 0
}

// relaxInput maps the pressed direction bits (1 = pressed) to what the chef is sent.
func (g *Game) relaxInput(bits byte) byte {
	h, v := byte(0), byte(0)
	switch bits & (InRight | InLeft) {
	case InRight:
		h = dirRight
	case InLeft:
		h = dirLeft
	}
	switch bits & (InUp | InDown) {
	case InUp:
		v = dirUp
	case InDown:
		v = dirDown
	}
	out := bits & InButton
	for _, d := range []byte{v, h} { // up / down first
		if d == 0 {
			continue
		}
		if u := g.usable(d); u != 0 {
			return out | dirBits(u)
		}
	}
	// nothing works: let the original decide with a single direction
	if v != 0 {
		return out | dirBits(v)
	}
	if h != 0 {
		return out | dirBits(h)
	}
	return out
}
