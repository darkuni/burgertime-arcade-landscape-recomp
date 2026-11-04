// Burger Time — a native Go rewrite of Data East's 1982 arcade game.
//
// The game needs the original ROM set (btimem.zip, or btime.zip if it holds the same files) for its graphics, level
// tables and sound data; none of Data East's data is bundled with this program.
//
// Controls:  arrows move, Z pepper, 5 insert coin, 1 / 2 start one / two player
// game, L cycle landscape / original square / cabinet 3:4,
// F11 toggle full screen, Esc quit.
//
// Gamepad: D-pad/left stick move, A or B pepper, Select coin, Start 1 player,
// X or Y 2 players, right bumper change view, left bumper toggle relaxed
// controls (R on the keyboard), click the right stick to switch Burger Time+
// (side-panel statistics) on or off (P on the keyboard), hold Select+Start to quit.
//
// Option -classic starts with the original square picture and no statistics.
package main

import (
	"errors"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"burgertime/internal/assets"
	"burgertime/internal/game"
	"burgertime/internal/sound"
	"burgertime/internal/stats"
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
	pad    padTracker
	// relaxed forgives near misses of the four-way joystick (R key / left bumper)
	relaxed   bool
	notice    int // frames left to show the controls message
	noticeMsg string
	// plus shows the Burger Time+ statistics panels in the landscape view
	plus     bool
	tr       *stats.Tracker
	trFrames int
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
	pd := a.pad.poll()
	if pd.quit {
		return ebiten.Termination
	}
	if pd.view {
		a.view = (a.view + 1) % 3
	}
	if pd.relaxed || inpututil.IsKeyJustPressed(ebiten.KeyR) {
		a.relaxed = !a.relaxed
		a.notice = 150
		a.noticeMsg = "CONTROLS: ORIGINAL"
		if a.relaxed {
			a.noticeMsg = "CONTROLS: RELAXED"
		}
		saveRelaxed(a.relaxed)
	}
	if pd.plus || inpututil.IsKeyJustPressed(ebiten.KeyP) {
		a.plus = !a.plus
		a.notice = 150
		a.noticeMsg = "BURGER TIME+: OFF (ORIGINAL)"
		if a.plus {
			a.noticeMsg = "BURGER TIME+: ON"
		}
		savePlus(a.plus)
	}
	a.g.Relaxed = a.relaxed
	if pd.left {
		p1 &^= game.InLeft
	}
	if pd.right {
		p1 &^= game.InRight
	}
	if pd.up {
		p1 &^= game.InUp
	}
	if pd.down {
		p1 &^= game.InDown
	}
	if pd.fire {
		p1 &^= game.InButton
	}
	sys := byte(0x07)
	if pd.start1 {
		sys &^= game.SysStart1
	}
	if pd.start2 {
		sys &^= game.SysStart2
	}
	if down(ebiten.Key1) {
		sys &^= game.SysStart1
	}
	if down(ebiten.Key2) {
		sys &^= game.SysStart2
	}
	coinPressed := inpututil.IsKeyJustPressed(ebiten.Key5) || pd.coin
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
		a.tr.Frame()
		for _, c := range a.g.SndCmds {
			a.synth.Command(c)
		}
		a.g.SndCmds = a.g.SndCmds[:0]
	}
	a.trFrames++
	if a.trFrames >= 600 && a.tr.Dirty {
		a.trFrames = 0
		a.saveStats()
	}
	return nil
}

func (a *app) Draw(screen *ebiten.Image) {
	defer a.drawNotice(screen)
	st := video.State{VRAM: a.g.VRAM, CRAM: a.g.CRAM, Palette: a.g.Palette, VidCtl: a.g.VidCtl}
	st.Scroll[0] = a.g.Scroll
	img := a.r.Image(&st)
	if a.view == 0 {
		if a.plus {
			video.LandscapePlus(a.land, img)
			stats.DrawPanels(a.land, a.tr)
		} else {
			video.Landscape(a.land, img)
		}
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
	var given []string
	for _, a := range os.Args[1:] {
		if !strings.HasPrefix(a, "-") {
			given = append(given, a)
		}
	}
	if len(given) > 0 {
		return given
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
	classic := false
	for _, arg := range os.Args[1:] {
		if arg == "-classic" || arg == "--classic" {
			classic = true
		}
	}
	a := &app{
		relaxed: loadRelaxed(),
		plus:    loadPlus() && !classic,
		g:       game.New(roms),
		r:       video.New(roms.Decode()),
		screen:  ebiten.NewImage(video.Visible, video.Visible),
		land:    image.NewRGBA(image.Rect(0, 0, video.LandscapeW, video.LandscapeH)),
		landIm:  ebiten.NewImage(video.LandscapeW, video.LandscapeH),
	}
	if classic {
		a.view = 1 // the original square picture
	}
	a.tr = stats.New(a.g)
	a.tr.Icons = stats.EnemyIcons(roms)
	if f := statsFile(); f != "" {
		a.tr.Load(f)
	}
	a.g.OnEvent = a.tr.Event
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
	a.saveStats()
}

// saveStats writes the all-time statistics.
func (a *app) saveStats() {
	if f := statsFile(); f != "" {
		_ = os.MkdirAll(filepath.Dir(f), 0o755)
		_ = a.tr.Save(f)
	}
}

func statsFile() string {
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "burgertime", "stats.json")
}

// Burger Time+ (the statistics panels) is on by default; the choice is remembered.
func plusFile() string {
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "burgertime", "plus.txt")
}

func loadPlus() bool {
	if f := plusFile(); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			return strings.TrimSpace(string(b)) != "classic"
		}
	}
	return true
}

func savePlus(on bool) {
	f := plusFile()
	if f == "" {
		return
	}
	v := "classic"
	if on {
		v = "plus"
	}
	_ = os.MkdirAll(filepath.Dir(f), 0o755)
	_ = os.WriteFile(f, []byte(v+"\n"), 0o644)
}

func (a *app) drawNotice(screen *ebiten.Image) {
	if a.notice <= 0 {
		return
	}
	a.notice--
	msg := a.noticeMsg
	x, y := 2, 2
	if a.view == 0 {
		x, y = 4, 190
	}
	ebitenutil.DebugPrintAt(screen, msg, x, y)
}

// The control style is remembered between runs.
func relaxedFile() string {
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "burgertime", "controls.txt")
}

func loadRelaxed() bool {
	if f := relaxedFile(); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			return strings.TrimSpace(string(b)) == "relaxed"
		}
	}
	return false
}

func saveRelaxed(on bool) {
	f := relaxedFile()
	if f == "" {
		return
	}
	v := "original"
	if on {
		v = "relaxed"
	}
	_ = os.MkdirAll(filepath.Dir(f), 0o755)
	_ = os.WriteFile(f, []byte(v+"\n"), 0o644)
}
