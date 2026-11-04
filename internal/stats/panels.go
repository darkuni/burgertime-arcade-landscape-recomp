package stats

import (
	"fmt"
	"image"
	"image/color"
)

var (
	colLabel = color.RGBA{73, 182, 182, 255}
	colValue = color.RGBA{255, 255, 255, 255}
	colHot   = color.RGBA{255, 255, 73, 255}
	colDim   = color.RGBA{110, 110, 140, 255}
)

func num(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func blitIcon(dst, ic *image.RGBA, x, y int) {
	b := ic.Bounds()
	for yy := 0; yy < b.Dy(); yy++ {
		for xx := 0; xx < b.Dx(); xx++ {
			c := ic.RGBAAt(xx, yy)
			if c.A != 0 && image.Pt(x+xx, y+yy).In(dst.Rect) {
				dst.SetRGBA(x+xx, y+yy, c)
			}
		}
	}
}

// Side panels of the 384x216 landscape picture: the maze occupies x 72..311.
const (
	leftX  = 7
	rightX = 313
)

// DrawPanels draws the Burger Time+ read-outs below the score boxes that the
// landscape layout already places in the side panels. The read-outs follow
// the player whose turn it is.
func DrawPanels(dst *image.RGBA, t *Tracker) {
	cur := t.Cur
	if !t.Playing {
		cur = 0
	}
	game, life := &t.GameP[cur], &t.LifeP[cur]

	// ---- left: this player's game ---------------------------------------
	y := 64
	if t.Playing && t.Players > 1 {
		DrawText(dst, leftX, y, fmt.Sprintf("PLAYER %d STATS", cur+1), colHot, 1)
		y += 8
	}
	row := func(label, value string, scale int) {
		DrawText(dst, leftX, y, label, colLabel, 1)
		DrawText(dst, leftX, y+7, value, colValue, scale)
		y += 8 + 5*scale + 4
	}
	level := t.Level[cur]
	if !t.Playing || level < 1 {
		level = 1
	}
	row("LEVEL", fmt.Sprint(level), 2)
	row("STEPS", num(game.Steps), 2)
	small := func(label, value string, c color.RGBA) {
		DrawText(dst, leftX, y, label, colDim, 1)
		DrawText(dst, leftX+TextWidth(label+" ", 1), y, value, c, 1)
		y += 7
	}
	small("PIECES", num(game.Pieces), colValue)
	small("BONUS", num(game.Bonus), colValue)
	small("SHOTS", num(game.Shots), colValue)
	y += 3
	DrawText(dst, leftX, y, "ALL TIME", colHot, 1)
	y += 8
	small("GAMES", num(life.Games), colValue)
	small("BEST LVL", num(life.BestLevel), colValue)
	small("STEPS", num(life.Steps), colValue)
	small("PIECES", num(life.Pieces), colValue)

	// ---- right: best drop, then the enemy tally --------------------------
	ry := 66
	DrawText(dst, rightX+4, ry, "BEST DROP", colLabel, 1)
	best := num(life.BestDrop)
	sc := 2
	for sc > 1 && TextWidth(best, sc) > 64 {
		sc--
	}
	DrawText(dst, rightX+4, ry+7, best, colHot, sc)
	DrawText(dst, rightX+4, ry+19, "GAME "+num(game.BestDrop), colDim, 1)
	ry += 32

	DrawText(dst, rightX+21, ry, "PEPPER", colValue, 1)
	DrawText(dst, rightX+49, ry, "CRUSH", colHot, 1)
	ry += 9
	for k := 0; k < Enemies; k++ {
		if ic := t.Icons[k]; ic != nil {
			blitIcon(dst, ic, rightX+2+(16-ic.Bounds().Dx())/2, ry+(14-ic.Bounds().Dy())/2)
		} else {
			DrawText(dst, rightX+8, ry+4, "?", colDim, 1)
		}
		DrawText(dst, rightX+22, ry, num(game.Hit[k]), colValue, 1)
		DrawText(dst, rightX+49, ry, num(game.Squash[k]), colHot, 1)
		DrawText(dst, rightX+22, ry+7, num(life.Hit[k]), colDim, 1)
		DrawText(dst, rightX+49, ry+7, num(life.Squash[k]), colDim, 1)
		ry += 20
	}
}
