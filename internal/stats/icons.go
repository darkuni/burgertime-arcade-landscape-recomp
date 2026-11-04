package stats

import (
	"image"
	"image/color"

	"burgertime/internal/assets"
	"burgertime/internal/game"
	"burgertime/internal/video"
)

// The first picture of each enemy in the ROM's frame lists: hot dog (kind 1),
// pickle (kind 2), egg (kind 3). The pictures are drawn from the player's own
// ROMs; nothing is bundled.
var enemySprites = [Enemies]byte{0x58, 0x64, 0x70}

// EnemyIcons draws the three enemy pictures. It runs a private, silent copy
// of the game up to the first level so that the palette is the one the game
// really uses.
func EnemyIcons(roms *assets.ROMs) [Enemies]*image.RGBA {
	g := game.New(roms)
	for f := 0; f < 500; f++ {
		sys := byte(0x07)
		if f == 60 {
			g.Coin()
		}
		if f >= 60 && f < 63 {
			sys |= 0x40
		}
		if f >= 100 && f < 104 {
			sys &^= 0x01
		}
		g.P1, g.P2, g.System = 0xff, 0xff, sys
		g.Frame()
		g.SndCmds = g.SndCmds[:0]
	}
	r := video.New(roms.Decode())
	var out [Enemies]*image.RGBA
	for k, code := range enemySprites {
		var st video.State
		st.Palette = g.Palette
		// sprite 0 at raw position (100, 99): upright (91, 132)
		st.VRAM[0], st.VRAM[0x20], st.VRAM[0x40], st.VRAM[0x60] = 1, code, 140, 140
		img := r.Image(&st)
		bg := img.RGBAAt(0, 0)
		ic := image.NewRGBA(image.Rect(0, 0, 16, 16))
		x0, y0, x1, y1 := 16, 16, -1, -1
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				c := img.RGBAAt(91+x, 132+y)
				if c == bg {
					continue
				}
				ic.SetRGBA(x, y, c)
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
		if x1 < 0 {
			continue
		}
		// crop to the drawing
		crop := image.NewRGBA(image.Rect(0, 0, x1-x0+1, y1-y0+1))
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				crop.SetRGBA(x-x0, y-y0, ic.RGBAAt(x, y))
			}
		}
		out[k] = crop
	}
	_ = color.RGBA{}
	return out
}
