package game

// drawBlockList draws every record of the level layout at $0202 (sub_cd3e).
// A record is four bytes: type, x, y and a spare byte; type 0 ends the list.
func (g *Game) drawBlockList() {
	for i := 0; i < 0x100; i += 4 {
		typ := g.RAM[0x202+i]
		if typ == 0 {
			return
		}
		g.drawBlock(typ, g.RAM[0x203+i], g.RAM[0x204+i], false)
	}
}

// drawBlockAuto is sub_cd7c: like drawBlock, but draws only the upper half when the
// block is on the top row of the screen.
func (g *Game) drawBlockAuto(typ, px, py byte) uint16 {
	return g.drawBlock(typ, px, py, py&0xf8 == 0)
}

// drawBlock is sub_cd73/cd8c: draws a burger ingredient (a 4x2 block of tiles) or the
// plate (type $0f) at the position px, py. The tile code, colour and shape come from
// the table at $ce91. oneRow draws only the upper row of tiles. It returns the
// address the original routine leaves in $05/$06 (the colour RAM address of the
// block, or the character address for the plate); some callers continue from it.
func (g *Game) drawBlock(typ, px, py byte, oneRow bool) uint16 {
	g.carry = false // the original's last arithmetic leaves the carry clear
	t := uint16(typ&0x0f) * 2
	tile := g.Rom(0xce91+t) + (py&7)<<2
	color := g.Rom(0xce92 + t)
	col := (^px) >> 3
	addr := 0x1000 + uint16(py&0xf8)<<2 + uint16(col)
	if typ&0x0f == 0x0f {
		// the plate: a different arrangement, starting one row up and one column left
		p := addr - 0x21
		g.Wr(p, tile)
		g.Wr(p+5, tile+2)
		tile++
		for y := uint16(0x21); y <= 0x24; y++ {
			g.Wr(p+y, tile)
			g.Wr(p+0x400+y, 0)
		}
		return addr
	}
	g.putRow(addr, tile)
	if !oneRow {
		g.putRow(addr+0x20, tile+0x20)
	}
	g.fillColors(addr+0x400, color)
	if !oneRow {
		g.fillColors(addr+0x420, color)
	}
	return addr + 0x400
}

// putRow stores four consecutive tile codes.
func (g *Game) putRow(addr uint16, tile byte) {
	for i := uint16(0); i < 4; i++ {
		g.Wr(addr+i, tile+byte(i))
	}
}

func (g *Game) fillColors(addr uint16, c byte) {
	for i := uint16(0); i < 4; i++ {
		g.Wr(addr+i, c)
	}
}
