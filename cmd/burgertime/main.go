// Burger Time — a native Go rewrite of Data East's 1982 arcade game.
//
// The game needs the original ROM set (btimem.zip, or btime.zip if it holds the same files) for its graphics, level
// tables and sound data; none of Data East's data is bundled with this program.
//
// Controls:  arrows move, Z pepper, 5 insert coin, 1 / 2 start one / two player
// game, L cycle landscape / original square / cabinet 3:4,
// F11 toggle full screen, Esc quit.
package main

import (
	"errors"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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
	// view: 0 landscape (default), 1 original square picture, 2 the cabinet's 3:4 shape
	view   int
	land   *image.RGBA
	landIm *ebiten.Image
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
		a.view = (a.view + 1) % 3
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
	if a.view == 0 {
		video.Landscape(a.land, img)
		a.landIm.WritePixels(a.land.Pix)
		screen.DrawImage(a.landIm, nil)
		return
	}
	a.screen.WritePixels(img.Pix)
	op := &ebiten.DrawImageOptions{}
	if a.view == 2 {
		op.GeoM.Scale(1, 4.0/3.0)
	}
	screen.DrawImage(a.screen, op)
}

func (a *app) Layout(_, _ int) (int, int) {
	switch a.view {
	case 0:
		return video.LandscapeW, video.LandscapeH
	case 2:
		return video.Visible, video.Visible * 4 / 3
	}
	return video.Visible, video.Visible
}

// findROMs returns every ROM set it can find: the path given on the command
// line, or btimem.zip / btime.zip (and a "roms" folder) next to the program,
// in the working directory, or in the home folder. Sets are merged by file name.
func findROMs() []string {
	if len(os.Args) > 1 {
		return os.Args[1:]
	}
	dirs := []string{"."}
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		dirs = append(dirs, d, filepath.Join(d, "roms"), filepath.Join(d, "..", "Resources"), filepath.Join(d, "..", "..", ".."))
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, home)
	}
	var out []string
	seen := map[string]bool{}
	for _, d := range dirs {
		for _, n := range []string{"btimem.zip", "btime.zip"} {
			c := filepath.Join(d, n)
			if abs, err := filepath.Abs(c); err == nil {
				c = abs
			}
			if seen[c] {
				continue
			}
			if _, err := os.Stat(c); err == nil {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out
}

// errApp shows a problem in the window, because the Windows build has no
// console to print to.
type errApp struct{ msg string }

func (e *errApp) Update() error { return nil }
func (e *errApp) Draw(s *ebiten.Image) {
	ebitenutil.DebugPrintAt(s, e.msg, 6, 6)
}
func (e *errApp) Layout(_, _ int) (int, int) { return 240, 240 }

func showError(msg string) {
	log.Print(msg)
	ebiten.SetWindowTitle("Burger Time - ROM problem")
	ebiten.SetWindowSize(240*3, 240*3)
	_ = ebiten.RunGame(&errApp{msg})
	os.Exit(1)
}

func romProblem(err error, paths []string) string {
	var me *assets.MissingError
	switch {
	case len(paths) == 0 && assets.Embedded == nil:
		return "ROM set not found.\n\nPut btimem.zip (or btime.zip)\nnext to burgertime.exe,\nor drag the zip onto the exe.\n\nThis port is built from the\nMidway set (btimem): it needs\nab05a1.12b, ab04.9b, ab06.13b,\nab05.10b, ab07.15b + the ab*\ngraphics/sound files."
	case errors.As(err, &me):
		m := "Found:\n"
		for _, p := range paths {
			m += "  " + filepath.Base(p) + "\n"
		}
		m += "\nbut these files are missing:\n"
		for i, n := range me.Missing {
			m += " " + n
			if i%3 == 2 {
				m += "\n"
			}
		}
		if me.DataEast {
			m += "\n\nThis looks like the Data East\nset (aa04.9b ...). The game logic\nhere comes from the Midway set\n(btimem), which has different\nprogram ROMs. Use btimem.zip\nor a merged set that has both."
		}
		return m
	}
	return fmt.Sprintf("Could not load the ROMs:\n%v", err)
}

func main() {
	paths := findROMs()
	var roms *assets.ROMs
	var err error
	if len(paths) == 0 && assets.Embedded != nil {
		roms, err = assets.LoadEmbedded()
	} else if len(paths) > 0 {
		roms, err = assets.LoadMulti(paths)
	} else {
		err = fmt.Errorf("no ROM set")
	}
	if err != nil {
		showError(romProblem(err, paths))
	}
	a := &app{
		g:      game.New(roms),
		r:      video.New(roms.Decode()),
		screen: ebiten.NewImage(video.Visible, video.Visible),
		land:   image.NewRGBA(image.Rect(0, 0, video.LandscapeW, video.LandscapeH)),
		landIm: ebiten.NewImage(video.LandscapeW, video.LandscapeH),
	}
	a.synth = sound.NewSynth(roms.Sound[:], sampleRate)
	ctx := audio.NewContext(sampleRate)
	player, err := ctx.NewPlayer(&pcm{s: a.synth})
	if err != nil {
		log.Fatal(err)
	}
	player.SetBufferSize(60 * time.Millisecond)
	player.Play()
	ebiten.SetWindowTitle("Burger Time")
	ebiten.SetWindowSize(video.LandscapeW*3, video.LandscapeH*3)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)
	if err := ebiten.RunGame(a); err != nil {
		log.Fatal(err)
	}
}
