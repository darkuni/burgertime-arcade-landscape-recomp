package video

import (
	"image"
	"image/color"
)

// Landscape layout: the maze (240 wide, the top score rows cut off) sits in the
// middle of a 384x216 picture and the score / pepper / lives / level read-outs
// move to the side panels, so the game fills a widescreen display.
const (
	LandscapeW = 384
	LandscapeH = 216
	sideW      = (LandscapeW - Visible) / 2
	fieldTop   = 16 // first row of the upright picture shown in the middle
)

type rect struct{ x, y, w, h int }

// Read-outs in the upright 240x240 picture.
var (
	hudPlayer  = rect{16, 0, 48, 16}  // 1UP / 2UP and the player's score
	hudHigh    = rect{64, 0, 64, 16}  // HI-SCORE
	hudPepper  = rect{192, 0, 44, 16} // PEPPER and its count
	hudLives   = rect{8, 208, 20, 16} // spare chefs, bottom left
	hudLevel   = rect{222, 216, 14, 8}
	hudPlayer2 = rect{146, 0, 46, 16} // 2UP and the second player's score
)

func blit(dst, src *image.RGBA, r rect, dx, dy int) {
	for y := 0; y < r.h; y++ {
		for x := 0; x < r.w; x++ {
			dst.SetRGBA(dx+x, dy+y, src.RGBAAt(r.x+x, r.y+y))
		}
	}
}

func fill(dst *image.RGBA, x0, y0, w, h int, c color.RGBA) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			dst.SetRGBA(x, y, c)
		}
	}
}

// Landscape builds the widescreen picture from the upright 240x240 one; dst
// must be LandscapeW x LandscapeH.
func Landscape(dst, up *image.RGBA) {
	black := color.RGBA{0, 0, 0, 255}
	fill(dst, 0, 0, LandscapeW, LandscapeH, black)
	blit(dst, up, rect{0, fieldTop, Visible, LandscapeH}, sideW, 0)
	// the lives and level icons come from the side panels, not the maze
	for _, r := range []rect{hudLives, hudLevel} {
		fill(dst, sideW+r.x, r.y-fieldTop, r.w, r.h, black)
	}
	line := color.RGBA{24, 36, 92, 255}
	fill(dst, sideW-1, 0, 1, LandscapeH, line)
	fill(dst, sideW+Visible, 0, 1, LandscapeH, line)
	rx := sideW + Visible + 1
	// left: player score and spare chefs
	blit(dst, up, hudPlayer, 12, 12)
	blit(dst, up, hudLives, 12, 40)
	// right: hi-score, pepper, level
	blit(dst, up, hudHigh, rx+2, 12)
	blit(dst, up, hudPepper, rx+14, 40)
	blit(dst, up, hudLevel, rx+22, 64)
}

// LandscapePlus is the layout for Burger Time+: both players' scores, the
// spare chefs and the pepper / level read-outs keep their places and the room
// below them is left for the statistics panels.
func LandscapePlus(dst, up *image.RGBA) {
	black := color.RGBA{0, 0, 0, 255}
	fill(dst, 0, 0, LandscapeW, LandscapeH, black)
	blit(dst, up, rect{0, fieldTop, Visible, LandscapeH}, sideW, 0)
	for _, r := range []rect{hudLives, hudLevel} {
		fill(dst, sideW+r.x, r.y-fieldTop, r.w, r.h, black)
	}
	line := color.RGBA{24, 36, 92, 255}
	fill(dst, sideW-1, 0, 1, LandscapeH, line)
	fill(dst, sideW+Visible, 0, 1, LandscapeH, line)
	rx := sideW + Visible + 1
	// left: both players' scores, spare chefs
	blit(dst, up, hudPlayer, 12, 6)
	blit(dst, up, hudPlayer2, 12, 24)
	blit(dst, up, hudLives, 12, 44)
	// right: hi-score, pepper, level
	blit(dst, up, hudHigh, rx+2, 6)
	blit(dst, up, hudPepper, rx+14, 28)
	blit(dst, up, hudLevel, rx+22, 50)
}
