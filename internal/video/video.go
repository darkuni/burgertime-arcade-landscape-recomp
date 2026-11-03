// Package video renders the Burger Time screen from video RAM, colour RAM and
// palette RAM using the decoded graphics.
package video

import (
	"image"
	"image/color"

	"burgertime/internal/assets"
)

// State is everything the video hardware reads.
type State struct {
	VRAM    [0x400]byte
	CRAM    [0x400]byte
	Palette [16]byte
	Scroll  [2]byte // $4004 writes (bit 4 of the first enables the background layer)
	VidCtl  byte    // $4002 writes (bit 0 flips the screen)
}

const (
	// Bitmap is the raw 256x256 raster before rotation.
	Bitmap = 256
	// Visible is the visible square of the bitmap, x and y 8..247.
	Visible = 240
)

// Renderer draws frames.
type Renderer struct {
	G    *assets.Gfx
	pal  [16]color.RGBA
	buf  [Bitmap * Bitmap]uint8 // palette index per pixel
	Flip bool
}

func New(g *assets.Gfx) *Renderer { return &Renderer{G: g} }

// DrawRaw paints the raw 256x256 bitmap (palette indices) from the state.
func (r *Renderer) DrawRaw(s *State) []uint8 {
	for i := range r.buf {
		r.buf[i] = 0
	}
	flip := s.VidCtl&1 != 0
	bgOn := s.Scroll[0]&0x10 != 0
	if bgOn {
		r.drawBackground(s, flip)
	}
	// characters
	for offs := 0; offs < 0x400; offs++ {
		x := 31 - offs/32
		y := offs % 32
		code := int(s.VRAM[offs]) + 256*int(s.CRAM[offs]&3)
		if flip {
			x = 31 - x
			y = 31 - y
		}
		r.blit8(code, x*8, y*8, flip, bgOn)
	}
	// sprites: 8 of them. The CPU sees them at $1800+4*i (through the transposing mirror),
	// which puts field k of sprite i at VRAM offset (4*i+k)*32 (row 0 of the tile map, off screen).
	for i := 0; i < 8; i++ {
		o := i * 0x80
		attr := s.VRAM[o]
		if attr&1 == 0 {
			continue
		}
		code := int(s.VRAM[o+0x20])
		y := 240 - int(s.VRAM[o+0x40])
		x := 240 - int(s.VRAM[o+0x60])
		fx := attr&4 != 0
		fy := attr&2 != 0
		if flip {
			x = 240 - x
			y = 240 - y
			fx, fy = !fx, !fy
		}
		y -= 1
		r.blit16(code, x, y, fx, fy)
		if flip {
			r.blit16(code, x, y-256, fx, fy)
		} else {
			r.blit16(code, x, y+256, fx, fy)
		}
	}
	return r.buf[:]
}

func (r *Renderer) blit8(code, px, py int, flip, over bool) {
	t := &r.G.Chars[code&1023]
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			sx, sy := x, y
			if flip {
				sx, sy = 7-x, 7-y
			}
			v := t[sy][sx]
			if v == 0 {
				continue
			}
			dx, dy := px+x, py+y
			if dx < 0 || dy < 0 || dx >= Bitmap || dy >= Bitmap {
				continue
			}
			r.buf[dy*Bitmap+dx] = v
		}
	}
}

func (r *Renderer) blit16(code, px, py int, fx, fy bool) {
	t := &r.G.Sprites[code&255]
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			sx, sy := x, y
			if fx {
				sx = 15 - x
			}
			if fy {
				sy = 15 - y
			}
			v := t[sy][sx]
			if v == 0 {
				continue
			}
			dx, dy := px+x, py+y
			if dx < 0 || dy < 0 || dx >= Bitmap || dy >= Bitmap {
				continue
			}
			r.buf[dy*Bitmap+dx] = v
		}
	}
}

// Image converts the raw bitmap to an upright 240x240 picture (ROT270).
func (r *Renderer) Image(s *State) *image.RGBA {
	raw := r.DrawRaw(s)
	for i := range r.pal {
		c := assets.Palette(s.Palette[i])
		r.pal[i] = color.RGBA{c.R, c.G, c.B, 255}
	}
	img := image.NewRGBA(image.Rect(0, 0, Visible, Visible))
	for ny := 0; ny < Visible; ny++ {
		for nx := 0; nx < Visible; nx++ {
			// new(nx,ny) = old(x = 8+Visible-1-ny, y = 8+nx)
			ox := 8 + Visible - 1 - ny
			oy := 8 + nx
			img.SetRGBA(nx, ny, r.pal[raw[oy*Bitmap+ox]])
		}
	}
	return img
}

// drawBackground paints the maze layer: four 256-pixel pages of 16x16 tiles, the
// page order and the horizontal position selected by the low bits written to $4004.
func (r *Renderer) drawBackground(s *State, flip bool) {
	scroll := -int(s.Scroll[0]&3) << 8
	start := 1
	if flip {
		start = 0
	}
	var tmap [4]int
	for i := range tmap {
		tmap[i] = start | int(s.Scroll[0]&4)
		start = (start + 1) & 3
	}
	for i := 0; i < 5; i, scroll = i+1, scroll+256 {
		page := tmap[i&3] * 0x100
		if scroll > 256 {
			break
		}
		if scroll < -256 {
			continue
		}
		for offs := 0; offs < 0x100; offs++ {
			x := 240 - (16*(offs/16) + scroll) - 1
			y := 16 * (offs % 16)
			if flip {
				x = 240 - x
				y = 240 - y
			}
			t := &r.G.BG[r.G.BGMap[page+offs]&63]
			for ty := 0; ty < 16; ty++ {
				for tx := 0; tx < 16; tx++ {
					sx, sy := tx, ty
					if flip {
						sx, sy = 15-tx, 15-ty
					}
					dx, dy := x+tx, y+ty
					if dx < 0 || dy < 0 || dx >= Bitmap || dy >= Bitmap {
						continue
					}
					r.buf[dy*Bitmap+dx] = 8 + t[sy][sx]
				}
			}
		}
	}
}
