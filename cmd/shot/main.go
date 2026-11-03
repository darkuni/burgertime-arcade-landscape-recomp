// shot runs the game without a window and writes PNG screenshots (development tool).
//
//	shot <rom> <outdir> frame[,frame...]
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"

	"burgertime/internal/assets"
	"burgertime/internal/game"
	"burgertime/internal/video"
)

func main() {
	roms, err := assets.Load(os.Args[1])
	if err != nil {
		panic(err)
	}
	g := game.New(roms)
	r := video.New(roms.Decode())
	want := map[int]bool{}
	last := 0
	for _, s := range strings.Split(os.Args[3], ",") {
		n, _ := strconv.Atoi(s)
		want[n] = true
		if n > last {
			last = n
		}
	}
	dir := byte(0xff)
	for f := 0; f <= last; f++ {
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
		if f > 400 {
			// walk about
			switch (f / 40) % 4 {
			case 0:
				dir = 0xff &^ game.InRight
			case 1:
				dir = 0xff &^ game.InUp
			case 2:
				dir = 0xff &^ game.InLeft
			default:
				dir = 0xff &^ game.InDown
			}
		}
		g.P1, g.P2, g.System = dir, 0xff, sys
		g.Frame()
		g.SndCmds = g.SndCmds[:0]
		if want[f] {
			st := video.State{VRAM: g.VRAM, CRAM: g.CRAM, Palette: g.Palette, VidCtl: g.VidCtl}
			st.Scroll[0] = g.Scroll
			img := r.Image(&st)
			fn := fmt.Sprintf("%s/f%04d.png", os.Args[2], f)
			fh, _ := os.Create(fn)
			png.Encode(fh, img)
			fh.Close()
			land := image.NewRGBA(image.Rect(0, 0, video.LandscapeW, video.LandscapeH))
			video.Landscape(land, img)
			fh, _ = os.Create(fmt.Sprintf("%s/f%04d_land.png", os.Args[2], f))
			png.Encode(fh, land)
			fh.Close()
		}
	}
}
