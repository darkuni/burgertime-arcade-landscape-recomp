package game

// sub_d046: the coin interrupt. It checks that a coin switch really is closed
// (the original samples it four times with a delay between the samples) and
// leaves a request in $fa that the vertical blank work picks up once the switch
// has opened again.
func (g *Game) sub_d046() {
	if g.RAM[zBooted] == 0 {
		return
	}
	g.RAM[0x02] = ^g.DSW2 & 0xe0 // coin mode switches
	if s := g.System & (SysCoin1 | SysCoin2); s != 0 {
		g.RAM[0x26] = s // which coin switch
		g.RAM[0xfa] = 1
	}
}
