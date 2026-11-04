package game

// Events let the host keep statistics without touching the game: when
// Game.OnEvent is non-nil the game reports what happens, and nothing about the
// game itself depends on it.

// EventKind says what happened.
type EventKind int

const (
	EvPepperShot EventKind = iota + 1 // the chef shook the pepper
	EvPepperHit                       // pepper stunned an enemy; Kind = enemy kind
	EvSquash                          // a burger piece squashed/crushed an enemy; Kind = enemy kind
	EvDrop                            // points paid for squashing enemies with a burger piece; Points
	EvPiece                           // a burger piece reached its plate
	EvBonus                           // the chef picked up a bonus item; Points
)

// Event is one thing that happened in the game.
type Event struct {
	Kind   EventKind
	Enemy  int // enemy kind 1..3 (hot dog, egg, pickle in the order the ROM numbers them)
	Points int
	Player int // 0 or 1: whose turn it was
}

func (g *Game) emit(k EventKind, enemy, points int) {
	if g.OnEvent != nil {
		g.OnEvent(Event{Kind: k, Enemy: enemy, Points: points, Player: int(g.RAM[zPlayer] & 1)})
	}
}

// scorePoints is the value of score table entry idx (see sub_ea8f).
func (g *Game) scorePoints(idx byte) int {
	bcd := func(v byte) int { return int(v>>4)*10 + int(v&15) }
	x := uint16(idx)
	return bcd(g.Rom(0xeb3b+x)) + bcd(g.Rom(0xeb46+x))*100 + bcd(g.Rom(0xeb51+x))*10000
}
