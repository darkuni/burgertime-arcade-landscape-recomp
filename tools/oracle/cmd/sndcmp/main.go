// sndcmp compares the native sound sequencer with the reference sound board:
// both are given the same commands at the same times and their AY register
// writes are compared (order and values exactly, time stamps by deviation).
//
//	sndcmp <rom dir> <total cycles> <cmd@cycle>...
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"burgertime/internal/assets"
	"burgertime/internal/sound"
	"burgertime/tools/oracle/machine"
)

type w struct {
	t         uint64
	chip      int
	reg, val  byte
}

func main() {
	rs, err := machine.LoadROMs(os.Args[1])
	if err != nil {
		panic(err)
	}
	m, err := machine.New(rs)
	if err != nil {
		panic(err)
	}
	s := m.Snd
	total, _ := strconv.ParseUint(os.Args[2], 0, 64)
	type ev struct {
		at  uint64
		cmd byte
	}
	var evs []ev
	for _, a := range os.Args[3:] {
		p := strings.Split(a, "@")
		c, _ := strconv.ParseUint(p[0], 0, 8)
		t, _ := strconv.ParseUint(p[1], 0, 64)
		evs = append(evs, ev{t, byte(c)})
	}
	s.CPU.Reset()
	line, lineCyc := 0, 0
	step := func() {
		c := s.CPU.Step()
		lineCyc += c
		for lineCyc >= 32 {
			lineCyc -= 32
			line = (line + 1) % 272
			s.SetDivider(line&8 != 0)
		}
	}
	for s.CPU.PC != 0xf027 {
		step()
	}
	base := s.CPU.Cycles
	// cycles until the next timer interrupt
	nmi := uint64(0)
	for k := 1; k < 700; k++ {
		l, p := (line+k)%272, (line+k-1)%272
		if l&8 != 0 && p&8 == 0 {
			nmi = uint64(k*32 - lineCyc)
			break
		}
	}
	s.Record = true
	s.Log = nil
	ei := 0
	var inject []uint64
	for s.CPU.Cycles-base < total {
		for ei < len(evs) && s.CPU.Cycles-base >= evs[ei].at {
			s.Command(evs[ei].cmd)
			inject = append(inject, s.CPU.Cycles-base)
			ei++
		}
		step()
	}
	var ref []w
	for _, x := range s.Log {
		ref = append(ref, w{x.Cycle - base, x.Chip, x.Reg, x.Val})
	}

	ar, err := assets.Load(os.Args[1])
	if err != nil {
		panic(err)
	}
	q := sound.New(ar.Sound[:])
	// the start-up writes of New are not part of the comparison
	var got []w
	q.OnAY = func(x sound.AYWrite) { got = append(got, w{x.Cycle, x.Chip, x.Reg, x.Val}) }
	q.SetNMIPhase(nmi)
	for i, e := range evs {
		if i >= len(inject) {
			break
		}
		if inject[i] > q.Now {
			q.Run(int(inject[i] - q.Now))
		}
		q.Command(e.cmd)
	}
	if total > q.Now {
		q.Run(int(total - q.Now))
	}
	maxd, bad := int64(0), 0
	for chip := 0; chip < 2; chip++ {
		var a, b []w
		for _, x := range ref {
			if x.chip == chip {
				a = append(a, x)
			}
		}
		for _, x := range got {
			if x.chip == chip {
				b = append(b, x)
			}
		}
		n := len(a)
		if len(b) < n {
			n = len(b)
		}
		if len(a) != len(b) {
			fmt.Printf("chip %d: ref %d writes, got %d\n", chip, len(a), len(b))
			bad++
		}
		for i := 0; i < n; i++ {
			if a[i].reg != b[i].reg || a[i].val != b[i].val {
				fmt.Printf("chip %d write %d differs: ref %v got %v\n", chip, i, a[i], b[i])
				bad++
				if bad > 8 {
					break
				}
				continue
			}
			d := int64(b[i].t) - int64(a[i].t)
			if d < 0 {
				d = -d
			}
			if d > maxd {
				maxd = d
			}
		}
	}
	n := len(ref)
	fmt.Printf("writes: ref %d got %d, first %d identical-or-checked, max time deviation %d cycles, injected %v\n", len(ref), len(got), n, maxd, inject)
	if len(got) > 0 && len(ref) > 0 && bad == 0 {
		fmt.Printf("last write: ref %d got %d\n", ref[len(ref)-1].t, got[len(got)-1].t)
	}
}
