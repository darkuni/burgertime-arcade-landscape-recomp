package main

import (
	"fmt"
	"os"

	"burgertime/tools/oracle/machine"
)

type st struct {
	pc uint16
	op byte
}

func run(mode int, rs machine.ROMSet, n int) []st {
	machine.ScrambleMode = mode
	m, _ := machine.New(rs)
	m.CPU.Reset()
	var out []st
	for i := 0; i < n; i++ {
		pc := m.CPU.PC
		m.RunCycles(1)
		out = append(out, st{pc, m.CPU.LastOp})
	}
	return out
}

func main() {
	rs, _ := machine.LoadROMs(os.Args[1])
	a := run(0, rs, 20000)
	for _, mode := range []int{1, 2, 3} {
		b := run(mode, rs, 20000)
		for i := range a {
			if a[i] != b[i] {
				fmt.Printf("mode0 vs mode%d diverge at step %d: m0 %04x op=%02x | m%d %04x op=%02x\n", mode, i, a[i].pc, a[i].op, mode, b[i].pc, b[i].op)
				for j := i - 6; j <= i+1; j++ {
					if j >= 0 {
						fmt.Printf("    m0 %04x %02x   m%d %04x %02x\n", a[j].pc, a[j].op, mode, b[j].pc, b[j].op)
					}
				}
				break
			}
		}
	}
}
