package game

// Player change and game over messages (sub_c282 and friends). The text records
// live in the program ROM at $c302 (PLAYER one), $c30d (PLAYER two), $c318 (READY)
// and $c325 (GAME OVER).
const (
	textPlayer1 = 0xc302
	textPlayer2 = 0xc30d
	textReady   = 0xc318
	textOver    = 0xc325
)

// sub_c282: "READY" for a single player game.
func (g *Game) sub_c282() {
	g.printText(textReady)
	g.readyPause()
}

// sub_c29d: "PLAYER ONE  READY".
func (g *Game) sub_c29d() {
	g.printText(textPlayer1)
	g.printText(textReady)
	g.readyPause()
}

// sub_c2b1: "PLAYER TWO  READY".
func (g *Game) sub_c2b1() {
	g.printText(textPlayer2)
	g.printText(textReady)
	g.readyPause()
}

// sub_c28d: jingle, then wait about half a second.
func (g *Game) readyPause() {
	g.sub_eb61(3)
	g.waitFrames(0x1f)
}

// sub_c2c5: "PLAYER ONE  GAME OVER", typed out slowly.
func (g *Game) sub_c2c5() {
	g.RAM[0xc8]++
	g.printText(textPlayer1)
	g.RAM[0xc8]++
	g.printText(textOver)
	g.waitFrames(0x1f)
}

// sub_c2dd: "PLAYER TWO  GAME OVER".
func (g *Game) sub_c2dd() {
	g.RAM[0xc8]++
	g.printText(textPlayer2)
	g.RAM[0xc8]++
	g.printText(textOver)
	g.waitFrames(0x1f)
}

// sub_c2f5: "GAME OVER".
func (g *Game) sub_c2f5() {
	g.RAM[0xc8]++
	g.printText(textOver)
	g.waitFrames(0x1f)
}
