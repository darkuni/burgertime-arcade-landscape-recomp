package assets

// Gfx is the decoded graphics: every pixel is a 3-bit colour index (0-7).
type Gfx struct {
	Chars   [1024][8][8]uint8  // 8x8 character tiles
	Sprites [256][16][16]uint8 // 16x16 sprites
	BG      [64][16][16]uint8  // 16x16 background tiles
	BGMap   [0x800]uint8       // background tile map: 8 pages of 16x16 tile numbers
}

// RGB is a decoded palette colour.
type RGB struct{ R, G, B uint8 }

// Decode builds the tile and sprite bitmaps. Planes are stored one per ROM;
// plane 0 is the most significant bit of the pixel value, and bit 7 of each
// byte is the leftmost pixel.
func (r *ROMs) Decode() *Gfx {
	g := &Gfx{}
	for t := 0; t < 1024; t++ {
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				var v uint8
				for p := 0; p < 3; p++ {
					v = v<<1 | r.Gfx[p*0x2000+t*8+y]>>(7-uint(x))&1
				}
				g.Chars[t][y][x] = v
			}
		}
	}
	dec16 := func(src []byte, plane, n int, dst *[16][16]uint8, t int) {}
	_ = dec16
	for t := 0; t < 256; t++ {
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				var v uint8
				for p := 0; p < 3; p++ {
					v = v<<1 | bit16(r.Gfx[p*0x2000:(p+1)*0x2000], t, x, y)
				}
				g.Sprites[t][y][x] = v
			}
		}
	}
	for t := 0; t < 64; t++ {
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				var v uint8
				for p := 0; p < 3; p++ {
					v = v<<1 | bit16(r.BGTiles[p*0x800:(p+1)*0x800], t, x, y)
				}
				g.BG[t][y][x] = v
			}
		}
	}
	copy(g.BGMap[:], r.BGMap[:])
	return g
}

// bit16 reads one pixel of a 16x16 tile from a single plane: 32 bytes per
// tile; the left eight columns are in the second 16 bytes.
func bit16(plane []byte, t, x, y int) uint8 {
	off := y
	if x < 8 {
		off += 16
	}
	return plane[t*32+off] >> (7 - uint(x&7)) & 1
}

// Palette decodes a hardware palette RAM byte (3-3-2 bits, inverted):
// bits 0-2 red, bits 3-5 green, bits 6-7 blue, all active low.
func Palette(v byte) RGB {
	v = ^v
	r := v & 7
	gr := v >> 3 & 7
	b := v >> 6 & 3
	return RGB{
		R: uint8(0x21*(r&1) + 0x47*(r>>1&1) + 0x97*(r>>2&1)),
		G: uint8(0x21*(gr&1) + 0x47*(gr>>1&1) + 0x97*(gr>>2&1)),
		B: uint8(0x52*(b&1) + 0xad*(b>>1&1)),
	}
}
