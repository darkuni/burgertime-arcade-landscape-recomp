// Burger Time — a native Go rewrite of Data East's 1982 arcade game.
//
// The game needs the original ROM set (btimem.zip) for its graphics, level
// tables and sound data; none of Data East's data is bundled with this program.
//
// Controls:  arrows move, Z pepper, 5 insert coin, 1 / 2 start one / two player
// game, L switch between square pixels and the cabinet's 3:4 picture,
// F11 toggle full screen, Esc quit.
package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"burgertime/internal/assets"
	"burgertime/internal/game"
	"burgertime/internal/sound"
	"burgertime/internal/video"
)

const (
	sampleRate = 48000
	// the board's vertical blank rate; the host runs at 60 updates per second
	gameHz = 57.444
	hostHz = 60.0
)

type app struct {
	g      *game.Game
	r      *video.Renderer
	screen *ebiten.Image
	synth  *sound.Synth
	acc    float64
	coin   int // frames left that the coin switch is shown as closed
	// cabinet: show the picture at the cabinet's 3:4 shape instead of square pixels
	cabinet bool
}

// pcm streams the synthesiser as 16-bit stereo little-endian samples.
type pcm struct {
	s   *sound.Synth
	buf []float32
}

func (p *pcm) Read(out []byte) (int, error) {
	n := len(out) / 4
	if cap(p.buf) < n {
		p.buf = make([]float32, n)
	}
	p.buf = p.buf[:n]
	p.s.Render(p.buf)
	for i, v := range p.buf {
		x := int16(v * 32767)
		out[4*i], out[4*i+1] = byte(x), byte(uint16(x)>>8)
		out[4*i+2], out[4*i+3] = out[4*i], out[4*i+1]
	}
	return n * 4, nil
}

func (a *app) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		a.cabinet = !a.cabinet
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	down := ebiten.IsKeyPressed
	p1 := byte(0xff)
	if down(ebiten.KeyRight) {
		p1 &^= game.InRight
	}
	if down(ebiten.KeyLeft) {
		p1 &^= game.InLeft
	}
	if down(ebiten.KeyUp) {
		p1 &^= game.InUp
	}
	if down(ebiten.KeyDown) {
		p1 &^= game.InDown
	}
	if down(ebiten.KeyZ) || down(ebiten.KeySpace) {
		p1 &^= game.InButton
	}
	sys := byte(0x07)
	if down(ebiten.Key1) {
		sys &^= game.SysStart1
	}
	if down(ebiten.Key2) {
		sys &^= game.SysStart2
	}
	coinPressed := inpututil.IsKeyJustPressed(ebiten.Key5)
	if coinPressed {
		a.coin = 3
	}
	if a.coin > 0 {
		sys |= game.SysCoin1
	}
	a.g.P1, a.g.P2, a.g.System = p1, p1, sys

	// the board runs at 57.44 frames per second, the host at 60 updates
	a.acc += gameHz / hostHz
	for a.acc >= 1 {
		a.acc--
		if coinPressed {
			a.g.Coin()
			coinPressed = false
		}
		if a.coin > 0 {
			a.coin--
		}
		a.g.Frame()
		for _, c := range a.g.SndCmds {
			a.synth.Command(c)
		}
		a.g.SndCmds = a.g.SndCmds[:0]
	}
	return nil
}

func (a *app) Draw(screen *ebiten.Image) {
	st := video.State{VRAM: a.g.VRAM, CRAM: a.g.CRAM, Palette: a.g.Palette, VidCtl: a.g.VidCtl}
	st.Scroll[0] = a.g.Scroll
	img := a.r.Image(&st)
	a.screen.WritePixels(img.Pix)
	op := &ebiten.DrawImageOptions{}
	if a.cabinet {
		op.GeoM.Scale(1, 4.0/3.0)
	}
	screen.DrawImage(a.screen, op)
}

func (a *app) Layout(_, _ int) (int, int) {
	if a.cabinet {
		return video.Visible, video.Visible * 4 / 3
	}
	return video.Visible, video.Visible
}

func findROM() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	cands := []string{"btimem.zip"}
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		cands = append(cands, filepath.Join(d, "btimem.zip"),
			filepath.Join(d, "..", "Resources", "btimem.zip"),
			filepath.Join(d, "..", "..", "..", "btimem.zip"))
		if home, err := os.UserHomeDir(); err == nil {
			cands = append(cands, filepath.Join(home, "btimem.zip"))
		}
	}
	for _, c := range cands {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func main() {
	path := findROM()
	if path == "" && assets.Embedded == nil {
		log.Fatal("btimem.zip not found: put it next to burgertime.exe or pass its path as the first argument")
	}
	var roms *assets.ROMs
	var err error
	if path == "" {
		roms, err = assets.LoadEmbedded()
	} else {
		roms, err = assets.Load(path)
	}
	if err != nil {
		log.Fatal(fmt.Errorf("loading %s: %w", path, err))
	}
	a := &app{
		g:      game.New(roms),
		r:      video.New(roms.Decode()),
		screen: ebiten.NewImage(video.Visible, video.Visible),
	}
	_ = image.Rect
	a.synth = sound.NewSynth(roms.Sound[:], sampleRate)
	ctx := audio.NewContext(sampleRate)
	player, err := ctx.NewPlayer(&pcm{s: a.synth})
	if err != nil {
		log.Fatal(err)
	}
	player.SetBufferSize(60 * time.Millisecond)
	player.Play()
	ebiten.SetWindowTitle("Burger Time")
	ebiten.SetWindowSize(video.Visible*3, video.Visible*3)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)
	if err := ebiten.RunGame(a); err != nil {
		log.Fatal(err)
	}
}
