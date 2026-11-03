// lockstep runs the Go rewrite (possibly still using the reference CPU for routines
// that are not ported yet) next to the reference machine and compares the complete
// game state at every start of vertical blank.
//
//	lockstep <rom dir or zip> <iterations>
//
// Environment: SEED (random input seed), NOCOIN, CONTINUE=1 (keep going after the first mismatch).
package main

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"

	"burgertime/internal/assets"
	"burgertime/internal/game"
	"burgertime/tools/oracle/hybrid"
	"burgertime/tools/oracle/machine"
)

// bytes of the work RAM that hold no game state: stack page and the padding
// stores that the original program makes to normalise its flags.
func masked(a int) bool {
	if a >= 0x100 && a < 0x200 {
		return true
	}
	if a >= 0x03 && a <= 0x0e { // pointer and loop temporaries of the helper routines
		return true
	}
	switch a {
	case 0xf5, 0xf6, 0xf7, 0xc7: // 0xc7: temporary that eb61 uses to preserve X
		return true
	}
	return false
}

func main() {
	rom := os.Args[1]
	iters, _ := strconv.Atoi(os.Args[2])
	seed := int64(1)
	if v := os.Getenv("SEED"); v != "" {
		seed, _ = strconv.ParseInt(v, 10, 64)
	}
	rs, err := machine.LoadROMs(rom)
	if err != nil {
		panic(err)
	}
	m, err := machine.New(rs)
	if err != nil {
		panic(err)
	}
	m.CPU.Reset()
	var lastW [0x800]uint16
	var osnd []byte
	var watch = -1
	if v := os.Getenv("WATCH"); v != "" {
		x, _ := strconv.ParseInt(v, 0, 32)
		watch = int(x)
	}
	curK := 0
	m.OnWrite = func(a uint16, v byte) {
		if a < 0x800 {
			lastW[a] = m.CPU.LastPC
		}
		if int(a) == watch {
			fmt.Printf("  [k=%d] oracle write $%04x=%02x at pc=%04x\n", curK, a, v, m.CPU.LastPC)
		}
	}
	m.SndWrite = func(v byte) { osnd = append(osnd, v) }
	m.NoSound = true
	cov := map[uint16]bool{}
	m.CPU.Fetch = func(pc uint16) { cov[pc] = true }

	ar, err := assets.Load(rom)
	if err != nil {
		panic(err)
	}
	g := game.New(ar)
	hy := hybrid.New()
	if os.Getenv("PURE") == "" {
		g.Legacy = hy // routines that are not ported run on the reference CPU
	}
	if v := os.Getenv("TRACEPC"); v != "" {
		x, _ := strconv.ParseInt(v, 0, 32)
		hy.Trace = func(pc uint16, a, xx, y byte) {
			if int(pc) == int(x) {
				fmt.Printf("  [go k=%d] pc=%04x A=%02x X=%02x Y=%02x\n", g.Frames, pc, a, xx, y)
			}
		}
	}

	syncPC := map[uint16]bool{0xc086: true, 0xcb05: true, 0xed77: true, 0xf482: true}
	runToSync := func() uint16 {
		for n := 0; ; n++ {
			if syncPC[m.CPU.PC] && n > 0 {
				return m.CPU.PC
			}
			m.RunCycles(1)
			if n > 20_000_000 {
				panic("oracle never reached a sync point")
			}
		}
	}

	rng := rand.New(rand.NewSource(seed))
	dir := byte(0xff)
	coinAt := map[int]bool{}
	startAt := map[int]byte{}
	if os.Getenv("NOCOIN") == "" {
		for _, f := range []int{120, 900, 1500, 2600, 4000} {
			coinAt[f] = true
		}
		startAt[300] = 0x01
		startAt[1700] = 0x02
		startAt[2800] = 0x01
	}
	// SCEN=1: a seeded random schedule with both coin slots, one and two player
	// starts, and a pause/tilt event; DIP=1 additionally randomises the DIP switches.
	tiltAt := -1
	if os.Getenv("SCEN") != "" {
		coinAt = map[int]bool{}
		startAt = map[int]byte{}
		t := 100
		for i := 0; i < 8; i++ {
			t += 50 + rng.Intn(900)
			if rng.Intn(3) == 0 {
				// two coins in a row
				coinAt[t] = true
				coinAt[t+12] = true
			} else {
				coinAt[t] = true
			}
			if rng.Intn(4) == 0 {
				coinAt[t+6] = true
			}
			s := byte(0x01)
			if rng.Intn(2) == 0 {
				s = 0x02
			}
			startAt[t+60+rng.Intn(80)] = s
		}
		if rng.Intn(2) == 0 {
			tiltAt = iters/2 + rng.Intn(iters/3+1)
		}
	}
	if os.Getenv("DIP") != "" {
		d1 := byte(0x30 | rng.Intn(16) | (rng.Intn(2) << 6))
		d2 := byte(rng.Intn(256))
		m.DSW1, m.DSW2 = d1, d2
		g.DSW1, g.DSW2 = d1, d2
		fmt.Printf("DIP: DSW1=%02x DSW2=%02x\n", d1, d2)
	}
	pokeEvery := 0
	if v := os.Getenv("POKE"); v != "" {
		pokeEvery, _ = strconv.Atoi(v)
	}
	coin2 := os.Getenv("SCEN") != ""
	cont := os.Getenv("CONTINUE") != ""
	bad := 0
	for k := 0; k < iters; k++ {
		curK = k
		pc := runToSync()
		_ = pc
		g.Frame()
		if k > 0 || true {
			if d := compare(m, g, osnd, &lastW); d != "" {
				fmt.Printf("iteration %d (oracle PC %04x): MISMATCH\n%s", k, pc, d)
				bad++
				if !cont || bad > 3 {
					os.Exit(1)
				}
			}
		}
		osnd = osnd[:0]
		g.SndCmds = g.SndCmds[:0]
		// inputs for the iteration that follows
		if k%20 == 0 {
			r := rng.Intn(7)
			dir = 0xff
			if r < 4 {
				dir &^= 1 << uint(r)
			}
			if rng.Intn(6) == 0 {
				dir &^= 0x10
			}
		}
		sys := byte(0x07)
		coin := false
		if coinAt[k] || coinAt[k-1] || coinAt[k-2] {
			if coin2 && (k/7)%3 == 0 {
				sys |= 0x80
			} else {
				sys |= 0x40
			}
		}
		if tiltAt >= 0 && k >= tiltAt && k < tiltAt+3 {
			sys &^= 0x04
		}
		if coinAt[k] {
			coin = true
		}
		for f, b := range startAt {
			if k >= f && k < f+4 {
				sys &^= b
			}
		}
		if os.Getenv("LIVES") != "" && m.RAM[0x1b] != 0 {
			// keep the chefs alive so that long games reach the later levels
			for _, a := range []int{0x29, 0x2a} {
				if m.RAM[a]&0x7f < 2 && m.RAM[a]&0x80 == 0 {
					m.RAM[a], g.RAM[a] = 3, 3
				}
			}
		}
		if os.Getenv("HISCORE") != "" && m.RAM[0x1b] != 0 {
			if k%600 == 0 {
				for a := 0x2d; a <= 0x32; a++ {
					v := byte(rng.Intn(10)<<4 | rng.Intn(10))
					if a%3 == 1 {
						v &= 0x0f
					}
					m.RAM[a], g.RAM[a] = v, v
				}
			}
			if k%900 == 450 {
				for _, a := range []int{0x29, 0x2a} {
					if m.RAM[a]&0x80 == 0 && m.RAM[a] > 1 {
						m.RAM[a], g.RAM[a] = 1, 1
					}
				}
			}
		}
		if pk := pokeEvery; pk > 0 && k%pk == 0 && m.RAM[0x1b] != 0 {
			// act as if the chef had just stepped on a whole ingredient (as sub_d99b does)
			var recs []int
			for a := 0x202; a < 0x300 && m.RAM[a] != 0; a += 4 {
				if n := m.RAM[a+4] & 0xf0; m.RAM[a]&0xf0 == 0 && m.RAM[a]&0x0f != 0x0f && (n == 0 || n == 0xf0) {
					recs = append(recs, a)
				}
			}
			if len(recs) > 0 {
				a := recs[rng.Intn(len(recs))]
				for _, ram := range [][]byte{m.RAM[:], g.RAM[:]} {
					ram[a] = ram[a]&0x0f | 0x10
					ram[a+2] = ram[a+2]&0xfc | 4
					ram[a+3] = 1
				}
			}
		}
		if pokeEvery > 0 && m.RAM[0x1b] != 0 {
			if m.RAM[0xc4] != 0 && m.RAM[0x60] == 0 && k%7 == 0 {
				// walk the chef onto the bonus item
				ty, tx := m.ROM[0xea75+int(m.RAM[0x63])], m.ROM[0xea7b+int(m.RAM[0x63])]
				m.VRAM[0x3c0], g.VRAM[0x3c0] = ty, ty
				m.VRAM[0x3e0], g.VRAM[0x3e0] = tx, tx
			}
			if k%500 == 0 {
				for a := 0x33; a <= 0x35; a++ {
					m.RAM[a], g.RAM[a] = 0, 0
				}
			}
			if k%300 == 0 {
				m.RAM[0x90], g.RAM[0x90] = 0x13, 0x13
			}
		}
		m.P1, m.P2, m.System = dir, 0xff, sys
		g.P1, g.P2, g.System = dir, 0xff, sys
		if coin {
			m.CPU.IRQ = true
			g.Coin()
		}
	}
	fmt.Printf("OK: %d iterations identical (game frames %d), %d distinct instructions executed\n", iters, g.Frames, len(cov))
	if p := os.Getenv("COVOUT"); p != "" {
		f, _ := os.Create(p)
		var l []int
		for pc := range cov {
			l = append(l, int(pc))
		}
		sort.Ints(l)
		for _, pc := range l {
			fmt.Fprintf(f, "%04x\n", pc)
		}
		f.Close()
	}
}

func compare(m *machine.Machine, g *game.Game, osnd []byte, lastW *[0x800]uint16) string {
	var out string
	n := 0
	add := func(s string) {
		if n < 24 {
			out += s
		}
		n++
	}
	var addrs []int
	for a := 0; a < 0x800; a++ {
		if !masked(a) && m.RAM[a] != g.RAM[a] {
			addrs = append(addrs, a)
		}
	}
	sort.Ints(addrs)
	for _, a := range addrs {
		add(fmt.Sprintf("  RAM $%04x oracle=%02x go=%02x (oracle last writer %04x)\n", a, m.RAM[a], g.RAM[a], lastW[a]))
	}
	for a := 0; a < 0x400; a++ {
		if m.VRAM[a] != g.VRAM[a] {
			add(fmt.Sprintf("  VRAM $%03x oracle=%02x go=%02x\n", a, m.VRAM[a], g.VRAM[a]))
		}
		if m.CRAM[a] != g.CRAM[a] {
			add(fmt.Sprintf("  CRAM $%03x oracle=%02x go=%02x\n", a, m.CRAM[a], g.CRAM[a]))
		}
	}
	for a := 0; a < 16; a++ {
		if m.Palette[a] != g.Palette[a] {
			add(fmt.Sprintf("  PAL %x oracle=%02x go=%02x\n", a, m.Palette[a], g.Palette[a]))
		}
	}
	if m.Scroll[0] != g.Scroll {
		add(fmt.Sprintf("  $4004 oracle=%02x go=%02x\n", m.Scroll[0], g.Scroll))
	}
	if m.VidCtl != g.VidCtl {
		add(fmt.Sprintf("  $4002 oracle=%02x go=%02x\n", m.VidCtl, g.VidCtl))
	}
	if fmt.Sprint(osnd) != fmt.Sprint(g.SndCmds) {
		add(fmt.Sprintf("  sound commands oracle=%v go=%v\n", osnd, g.SndCmds))
	}
	if n > 24 {
		out += fmt.Sprintf("  ... %d differences in total\n", n)
	}
	return out
}
