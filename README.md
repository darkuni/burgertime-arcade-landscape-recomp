# Burger Time — native Go rewrite

A rewrite (not an emulator) of Data East's 1982 Burger Time in Go + Ebitengine.
The game logic, sound sequencer and AY synthesiser are Go code; the original ROM
set (`btimem.zip`) is only read for graphics, level tables and sound data.

Every routine of the main program was checked frame by frame against a reference
machine (dev-only, in `tools/oracle`) over hundreds of thousands of frames,
including coins, 1/2 players, DIP settings, level changes and high-score entry.
The sound sequencer produces the same AY register writes as the original sound CPU program.

## Files
- `burgertime.exe` – needs `btimem.zip` next to it (or pass its path as an argument)
- `burgertime-standalone.exe` – ROM compiled in; personal use only, do not distribute
- `burgertime-src.zip` – source

## Controls
Arrows move, Z (or Space) pepper, 5 coin, 1 / 2 start, L square pixels / cabinet 3:4 shape, F11 full screen, Esc quit.

## Releases

Publishing a GitHub release runs `.github/workflows/release.yml`, which builds
`burgertime.exe` (Windows) and `BurgerTime-mac.zip` (universal macOS app) and attaches
them to the release. Neither contains the ROM: put your own `btimem.zip` next to the game.
