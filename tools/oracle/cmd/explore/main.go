// explore runs the reference machine with scripted/random input and records
// executed code (with the opcode as actually decoded), RAM access counts and
// I/O port writes. Output feeds the annotated disassembly.
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"

	"burgertime/tools/oracle/machine"
)

type covEnt struct {
	Op    byte `json:"op"`
	Raw   byte `json:"raw"`
	Count int  `json:"n"`
}

func main() {
	rom := os.Args[1]
	frames, _ := strconv.Atoi(os.Args[2])
	out := os.Args[3]
	seed := int64(1)
	if v := os.Getenv("SEED"); v != "" {
		seed, _ = strconv.ParseInt(v, 10, 64)
	}
	rs, err := machine.LoadROMs(rom)
	if err != nil {
		panic(err)
	}
	m, _ := machine.New(rs)
	if v := os.Getenv("DSW2"); v != "" {
		x, _ := strconv.ParseInt(v, 0, 32)
		m.DSW2 = byte(x)
	}
	m.CPU.Reset()
	rng := rand.New(rand.NewSource(seed))
	cov := map[uint16]*covEnt{}
	m.CPU.Fetch = func(pc uint16) {}
	ports := map[string]int{}
	m.OnWrite = func(a uint16, v byte) {
		if a >= 0x4000 && a < 0x4005 {
			ports[fmt.Sprintf("%04x=%02x", a, v)]++
		}
	}
	var ramR, ramW [0x800]int
	prevR := m.OnRead
	_ = prevR
	m.OnRead = func(a uint16, v byte) {
		if a < 0x800 {
			ramR[a]++
		}
	}
	w := m.OnWrite
	m.OnWrite = func(a uint16, v byte) {
		w(a, v)
		if a < 0x800 {
			ramW[a]++
		}
	}
	m.Trace = nil
	// we capture coverage after each step through LastPC/LastOp
	step := func() {
		pc := m.CPU.PC
		m.RunCycles(1)
		e := cov[pc]
		if e == nil {
			e = &covEnt{Op: m.CPU.LastOp, Raw: m.ROM[pc]}
			cov[pc] = e
		}
		e.Count++
	}
	coinAt := map[int]bool{}
	startAt := map[int]byte{}
	if os.Getenv("NOINPUT") == "" {
		for _, f := range []int{120, 900, 1500} {
			coinAt[f] = true
		}
		startAt[300] = 0x01 // 1P start
		startAt[1700] = 0x02
	}
	dir := byte(0xff)
	for f := 0; f < frames; f++ {
		// input for this frame
		if coinAt[f] {
			m.System |= 0x40
			m.CPU.IRQ = true
		}
		if coinAt[f-3] {
			m.System &^= 0x40
		}
		if b, ok := startAt[f]; ok {
			m.System &^= b
		}
		if _, ok := startAt[f-4]; ok {
			m.System |= 0x03
		}
		if f%20 == 0 {
			r := rng.Intn(7)
			dir = 0xff
			switch r {
			case 0:
				dir &^= 0x01
			case 1:
				dir &^= 0x02
			case 2:
				dir &^= 0x04
			case 3:
				dir &^= 0x08
			}
			if rng.Intn(6) == 0 {
				dir &^= 0x10
			}
		}
		m.P1 = dir
		target := m.CPU.Cycles + machine.FrameCycles
		for m.CPU.Cycles < target {
			step()
		}
	}
	type rec struct {
		PC uint16 `json:"pc"`
		covEnt
	}
	var pcs []int
	for pc := range cov {
		pcs = append(pcs, int(pc))
	}
	sort.Ints(pcs)
	var list []rec
	for _, pc := range pcs {
		list = append(list, rec{uint16(pc), *cov[uint16(pc)]})
	}
	b, _ := json.Marshal(list)
	os.WriteFile(out, b, 0644)
	fmt.Printf("frames=%d covered=%d instrs illegal=%d\n", frames, len(cov), m.CPU.Illegal)
	var ks []string
	for k := range ports {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		fmt.Printf("port write %s x%d\n", k, ports[k])
	}
	var nr, nw int
	for i := range ramR {
		if ramR[i] > 0 {
			nr++
		}
		if ramW[i] > 0 {
			nw++
		}
	}
	fmt.Printf("RAM bytes read=%d written=%d\n", nr, nw)
	rb, _ := json.Marshal(map[string]any{"r": ramR[:], "w": ramW[:]})
	os.WriteFile(out+".ram.json", rb, 0644)
}
