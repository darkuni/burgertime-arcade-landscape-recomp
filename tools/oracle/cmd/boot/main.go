package main

import (
	"fmt"
	"os"
	"sort"

	"burgertime/tools/oracle/machine"
)

var dumped bool

func main() {
	if v := os.Getenv("SMODE"); v != "" {
		fmt.Sscan(v, &machine.ScrambleMode)
	}
	rs, err := machine.LoadROMs(os.Args[1])
	if err != nil {
		panic(err)
	}
	m, err := machine.New(rs)
	if err != nil {
		panic(err)
	}
	m.WriteLog = map[uint16]int{}
	m.CPU.Reset()
	fmt.Printf("reset PC=%04x\n", m.CPU.PC)
	seen := map[uint16]bool{}
	m.Trace = func(pc uint16) {
		if len(seen) < 40 && !seen[pc] {
			fmt.Printf("%04x ", pc)
		}
		seen[pc] = true
	}
	type ent struct {
		pc      uint16
		op      byte
		a, x, y byte
		p       byte
	}
	if os.Getenv("CALLS") != "" {
		n := 0
		m.Trace = func(pc uint16) {
			if n < 400 && m.CPU.LastOp == 0x20 && m.CPU.LastPC != 0 {
			}
		}
		last := uint16(0)
		cnt := 0
		for m.Frame < 300 && cnt < 150 {
			pc := m.CPU.PC
			m.RunCycles(1)
			if m.CPU.LastPC == pc && m.CPU.LastOp == 0x20 {
				t := m.CPU.PC
				if t != last {
					fmt.Printf("f%d %04x->JSR %04x\n", m.Frame, pc, t)
					cnt++
				}
				last = t
			}
		}
		os.Exit(0)
	}
	if os.Getenv("AFTER") != "" {
		started := false
		n := 0
		for n < 160 && m.Frame < 50 {
			pc := m.CPU.PC
			if pc == 0xd36c {
				started = true
			}
			m.RunCycles(1)
			if started {
				fmt.Printf("%04x:%02x A=%02x X=%02x Y=%02x P=%02x SP=%02x\n", pc, m.CPU.LastOp, m.CPU.A, m.CPU.X, m.CPU.Y, m.CPU.P, m.CPU.SP)
				n++
			}
		}
		os.Exit(0)
	}
	if os.Getenv("AMBIG") != "" {
		machine.OnDescramble = func(pc uint16, raw, dec byte) {
			if raw != dec {
				fmt.Printf("descrambled %04x raw=%02x -> %02x   (plain would be %02x)\n", pc, raw, dec, raw)
			}
		}
		seen := map[uint16]bool{}
		_ = seen
		m.RunCycles(machine.FrameCycles * 3)
		os.Exit(0)
	}
	if v := os.Getenv("DIS"); v != "" {
		m.Trace = nil
		var n int
		fmt.Sscan(v, &n)
		for i := 0; i < n; i++ {
			pc := m.CPU.PC
			m.RunCycles(1)
			fmt.Printf("%04x %02x %02x %02x\n", pc, m.CPU.LastOp, m.ROM[pc+1], m.ROM[pc+2])
		}
		os.Exit(0)
	}
	if os.Getenv("EARLY") != "" {
		n := 0
		m.Trace = func(pc uint16) {
			if n < 60 {
				fmt.Printf("%04x ", pc)
			}
			n++
		}
		m.RunCycles(3000)
		fmt.Println()
		os.Exit(0)
	}
	var hist [64]ent
	hn := 0
	m.Trace = func(pc uint16) {
		if hn > 0 {
			hist[(hn-1)&63].op = m.CPU.LastOp
		}
		hist[hn&63] = ent{pc: pc, a: m.CPU.A, x: m.CPU.X, y: m.CPU.Y, p: m.CPU.P}
		hn++
		if m.CPU.Illegal > 0 && !dumped {
			dumped = true
			for i := 0; i < 64; i++ {
				e := hist[(hn+i)&63]
				fmt.Printf("%04x op=%02x A=%02x X=%02x Y=%02x P=%02x\n", e.pc, e.op, e.a, e.x, e.y, e.p)
			}
			fmt.Printf("\nFIRST ILLEGAL at %04x\n", pc)
			fmt.Printf("A=%02x X=%02x Y=%02x SP=%02x P=%02x zp0c=%02x%02x\n", m.CPU.A, m.CPU.X, m.CPU.Y, m.CPU.SP, m.CPU.P, m.RAM[0xd], m.RAM[0xc])
		}
	}
	m.RunCycles(machine.FrameCycles * 200)
	fmt.Printf("\nframes=%d PC=%04x illegal=%d halted=%v\n", m.Frame, m.CPU.PC, m.CPU.Illegal, m.CPU.Halted)
	var ks []int
	for k := range m.WriteLog {
		ks = append(ks, int(k))
	}
	sort.Ints(ks)
	for _, k := range ks {
		fmt.Printf("unmapped write %04x x%d\n", k, m.WriteLog[uint16(k)])
	}
	fmt.Println("palette:", m.Palette)
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			fmt.Printf("%02x", m.VRAM[y*32+x])
		}
		fmt.Println()
	}
}
