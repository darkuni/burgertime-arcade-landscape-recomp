# Burger Time — native Go rewrite

A rewrite (not an emulator) of Data East's 1982 Burger Time in Go + Ebitengine.
The game logic, sound sequencer and AY synthesiser are Go code; the original ROM
set (`btimem.zip`) is only read for graphics, level tables and sound data.

Every routine of the main program was checked frame by frame against a reference
machine (dev-only, in `tools/oracle`) over hundreds of thousands of frames,
including coins, 1/2 players, DIP settings, level changes and high-score entry.
The sound sequencer produces the same AY register writes as the original sound CPU program.

## ROMs (required, not included)

You need your own copy of the MAME **Burger Time (Midway)** set, `btimem`. MAME ships it
*split*, so you need **both** zips, in the same folder as the game:

- `btimem.zip` - the Midway program ROMs (`ab05a1.12b`, `ab04.9b`, `ab06.13b`, `ab05.10b`, `ab07.15b`) plus `ab8.13k`, `ab12.7k`
- `btime.zip` - the parent set; supplies the graphics, sound and background ROMs (`ab00.1b`-`ab03.6b`, `ab9.15k`, `ab10.10k`, `ab11.12k`, `ab13.9k`, `ab14.12h`)

A full (non-split) `btimem.zip` or a merged set also works. The game looks next to the
executable, in a `roms` folder there, in the working directory and in your home folder
(on macOS also inside the `.app`), or you can pass zip paths as arguments. If something
is missing the game opens a window listing the missing files.

`btime.zip` alone is not enough: the Data East program ROMs (`aa04.9b`, ...) are a different
version of the game and are not supported yet.

## Status

- Game logic, sound and controls: complete, verified against the original (see above).
- Landscape layout: done (default). The maze sits in the middle of a 16:9 picture and the score, hi-score, pepper, spare chefs and level icons move to the side panels. L cycles landscape / original square / the cabinet's 3:4 shape.
- Builds: Windows, Linux x86_64 (Steam Deck / SteamOS desktop mode), macOS. Only Windows and the
  ROM loader have been tested on real hardware so far.

## Controls
Arrows move, Z (or Space) pepper, 5 coin, 1 / 2 start, L cycle landscape / square / cabinet 3:4, F11 full screen, Esc quit.

## Releases

Publishing a GitHub release runs `.github/workflows/release.yml`, which builds
`burgertime.exe` (Windows), `BurgerTime-linux.tar.gz` (Linux x86_64, runs on Steam Deck / SteamOS desktop mode) and `BurgerTime-mac.zip` (universal macOS app) and attaches
them to the release. None of them contain the ROMs (see above).
