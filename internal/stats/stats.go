// Package stats keeps the Burger Time+ statistics and draws them in the side
// panels. It only listens to the game (game.Event and the game's memory); it
// never changes it.
package stats

import (
	"encoding/json"
	"image"
	"os"

	"burgertime/internal/game"
)

// Enemy kinds as the ROM numbers them (1..3); index = kind-1.
const Enemies = 3

// Counts are one set of totals (this game, or all games) for one player.
type Counts struct {
	Games     int
	Steps     int
	BestLevel int
	Pieces    int // burger pieces that reached their plate
	Bonus     int // bonus items picked up
	Shots     int // pepper shakes
	Hit       [Enemies]int
	Squash    [Enemies]int
	BestDrop  int // most points one burger drop has paid for squashed enemies
}

// Zero page variables of the main program that the tracker reads.
const (
	zPlaying = 0x1b
	zPlayer  = 0x1f
	zTwoPlay = 0x21
	zLevel   = 0x61 // + player
	hiScore  = 0x33 // 3 BCD bytes, low first
	chefY    = 0x181e
	chefX    = 0x181f
)

// stepPixels is how far the chef walks for one counted step (one maze tile).
const stepPixels = 8

type Tracker struct {
	GameP, LifeP [2]Counts
	Players      int // 1 or 2 players in the running game
	Cur          int // whose turn it is
	Playing      bool
	Level        [2]int
	High         int
	Icons        [Enemies]*image.RGBA // hot dog, pickle, egg
	Dirty        bool

	g       *game.Game
	havePos bool
	px, py  int
	dist    int
}

func New(g *game.Game) *Tracker {
	return &Tracker{g: g, Players: 1}
}

func bcd(v byte) int { return int(v>>4)*10 + int(v&15) }

// Game and Life are the current player's totals for this game and for all
// games.
func (t *Tracker) Game() *Counts { return &t.GameP[t.Cur] }
func (t *Tracker) Life() *Counts { return &t.LifeP[t.Cur] }

// Event is the game's event hook.
func (t *Tracker) Event(e game.Event) {
	if !t.Playing || e.Player > 1 {
		return
	}
	g, l := &t.GameP[e.Player], &t.LifeP[e.Player]
	k := e.Enemy - 1
	switch e.Kind {
	case game.EvPepperShot:
		g.Shots++
		l.Shots++
	case game.EvPepperHit:
		if k >= 0 && k < Enemies {
			g.Hit[k]++
			l.Hit[k]++
		}
	case game.EvSquash:
		if k >= 0 && k < Enemies {
			g.Squash[k]++
			l.Squash[k]++
		}
	case game.EvDrop:
		if e.Points > g.BestDrop {
			g.BestDrop = e.Points
		}
		if e.Points > l.BestDrop {
			l.BestDrop = e.Points
		}
	case game.EvPiece:
		g.Pieces++
		l.Pieces++
	case game.EvBonus:
		g.Bonus++
		l.Bonus++
	default:
		return
	}
	t.Dirty = true
}

// Frame is called once per emulated frame, after the game has run.
func (t *Tracker) Frame() {
	R := &t.g.RAM
	t.High = bcd(R[hiScore+2])*10000 + bcd(R[hiScore+1])*100 + bcd(R[hiScore])
	playing := R[zPlaying] != 0
	if playing && !t.Playing {
		// a new game
		t.GameP = [2]Counts{}
		t.Players = 1
		t.Cur = 0
		t.LifeP[0].Games++
		t.GameP[0].Games = 1
		if R[zTwoPlay] != 0 {
			t.Players = 2
			t.LifeP[1].Games++
			t.GameP[1].Games = 1
		}
		t.havePos = false
		t.Dirty = true
	}
	t.Playing = playing
	if !playing {
		return
	}
	t.Cur = int(R[zPlayer] & 1)
	for p := 0; p < 2; p++ {
		t.Level[p] = int(R[zLevel+p])
		if t.Level[p] > t.GameP[p].BestLevel {
			t.GameP[p].BestLevel = t.Level[p]
		}
		if t.Level[p] > t.LifeP[p].BestLevel {
			t.LifeP[p].BestLevel = t.Level[p]
			t.Dirty = true
		}
	}
	// steps: distance walked by the chef, in maze tiles
	x, y := int(t.g.Rd(chefX)), int(t.g.Rd(chefY))
	if t.havePos {
		d := abs(x-t.px) + abs(y-t.py)
		if d > 0 && d <= 12 { // bigger jumps are a new life or level
			t.dist += d
			for t.dist >= stepPixels {
				t.dist -= stepPixels
				t.Game().Steps++
				t.Life().Steps++
			}
		}
	}
	t.px, t.py, t.havePos = x, y, true
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// ---- saving ------------------------------------------------------------

type saved struct {
	Life  Counts // player 1
	Life2 Counts // player 2
}

// Load reads the all-time totals from path (a missing file is fine).
func (t *Tracker) Load(path string) {
	d, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var s saved
	if json.Unmarshal(d, &s) != nil {
		return
	}
	t.LifeP[0], t.LifeP[1] = s.Life, s.Life2
}

// Save writes the all-time totals to path.
func (t *Tracker) Save(path string) error {
	d, err := json.MarshalIndent(saved{Life: t.LifeP[0], Life2: t.LifeP[1]}, "", " ")
	if err != nil {
		return err
	}
	t.Dirty = false
	return os.WriteFile(path, d, 0o644)
}
