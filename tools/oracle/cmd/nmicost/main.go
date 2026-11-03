package main

import (
	"fmt"
	"os"
	"sort"

	"burgertime/tools/oracle/machine"
)

func main() {
	rs, _ := machine.LoadROMs(os.Args[1])
	m, _ := machine.New(rs)
	s := m.Snd
	s.CPU.Reset()
	line, lineCyc := 0, 0
	var inNMI bool
	var start uint64
	var phaseSeen byte
	cost := map[string]map[uint64]int{}
	step := func() {
		pc := s.CPU.PC
		if pc == 0xf000 {
			inNMI = true
			start = s.CPU.Cycles - 7
			phaseSeen = s.RAM[4]
		}
		c := s.CPU.Step()
		if inNMI && pc == 0xf079 {
			inNMI = false
			k := fmt.Sprintf("phase%d en=%v", phaseSeen, s.RAM[1] != 0)
			if cost[k] == nil {
				cost[k] = map[uint64]int{}
			}
			cost[k][s.CPU.Cycles-start]++
		}
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
	s.Command(3)
	for s.CPU.Cycles-base < 3000000 {
		step()
	}
	var ks []string
	for k := range cost {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		fmt.Println(k, cost[k])
	}
}
