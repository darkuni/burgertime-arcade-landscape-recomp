package stats

import (
	"image"
	"path/filepath"
	"testing"

	"burgertime/internal/assets"
	"burgertime/internal/game"
)

func newTracker() *Tracker {
	g := game.New(&assets.ROMs{})
	g.RAM[zPlaying] = 1
	t := New(g)
	t.Playing = true
	return t
}

func TestEventsAreCountedPerPlayer(t *testing.T) {
	tr := newTracker()
	tr.Event(game.Event{Kind: game.EvPepperHit, Enemy: 1, Player: 0})
	tr.Event(game.Event{Kind: game.EvSquash, Enemy: 3, Player: 1})
	tr.Event(game.Event{Kind: game.EvSquash, Enemy: 3, Player: 1})
	tr.Event(game.Event{Kind: game.EvDrop, Points: 4000, Player: 1})
	tr.Event(game.Event{Kind: game.EvDrop, Points: 300, Player: 1})
	tr.Event(game.Event{Kind: game.EvPiece, Player: 0})
	tr.Event(game.Event{Kind: game.EvBonus, Player: 0})
	tr.Event(game.Event{Kind: game.EvPepperShot, Player: 0})
	if tr.GameP[0].Hit[0] != 1 || tr.GameP[1].Hit[0] != 0 {
		t.Fatal("pepper hit belongs to player 1")
	}
	if tr.GameP[1].Squash[2] != 2 || tr.LifeP[1].Squash[2] != 2 || tr.GameP[0].Squash[2] != 0 {
		t.Fatal("squashes belong to player 2")
	}
	if tr.GameP[1].BestDrop != 4000 {
		t.Fatalf("best drop %d", tr.GameP[1].BestDrop)
	}
	if tr.GameP[0].Pieces != 1 || tr.GameP[0].Bonus != 1 || tr.GameP[0].Shots != 1 {
		t.Fatal("piece, bonus, shot")
	}
}

func TestNothingCountedOutsideAGame(t *testing.T) {
	tr := newTracker()
	tr.Playing = false // attract-mode demo
	tr.Event(game.Event{Kind: game.EvSquash, Enemy: 1})
	if tr.GameP[0].Squash[0] != 0 {
		t.Fatal("demo must not count")
	}
}

func TestNewGameResetsTheGameTotals(t *testing.T) {
	tr := newTracker()
	tr.Playing = false
	tr.GameP[0].Pieces = 9
	tr.g.RAM[zTwoPlay] = 1
	tr.Frame()
	if tr.GameP[0].Pieces != 0 || tr.Players != 2 || tr.LifeP[0].Games != 1 || tr.LifeP[1].Games != 1 {
		t.Fatalf("new two player game: %+v", tr)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	tr := newTracker()
	tr.LifeP[0] = Counts{Games: 3, Pieces: 12, BestDrop: 8000}
	tr.LifeP[1].Squash[1] = 7
	f := filepath.Join(t.TempDir(), "stats.json")
	if err := tr.Save(f); err != nil {
		t.Fatal(err)
	}
	u := newTracker()
	u.Load(f)
	if u.LifeP != tr.LifeP {
		t.Fatalf("totals differ: %+v vs %+v", u.LifeP, tr.LifeP)
	}
}

func TestDrawPanelsDoesNotPanic(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 384, 216))
	tr := newTracker()
	tr.Players, tr.Cur = 2, 1
	tr.GameP[1].Steps = 1234567
	tr.Icons[0] = image.NewRGBA(image.Rect(0, 0, 13, 15))
	DrawPanels(dst, tr)
	if num(1234567) != "1,234,567" {
		t.Fatal("number formatting")
	}
}
