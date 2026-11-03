// k6502 runs Klaus Dormann's 6502 functional test against the reference CPU.
package main

import (
	"fmt"
	"os"

	"burgertime/tools/oracle/m6502"
)

type ram [65536]byte

func (r *ram) Read(a uint16) byte     { return r[a] }
func (r *ram) Write(a uint16, v byte) { r[a] = v }

func main() {
	var m ram
	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	copy(m[:], b)
	c := &m6502.CPU{Bus: &m}
	c.PC = 0x400
	c.SP = 0xfd
	c.P = 0x24
	var last uint16 = 0xffff
	same := 0
	for i := 0; i < 200_000_000; i++ {
		pc := c.PC
		c.Step()
		if c.Halted {
			fmt.Printf("HALT at %04x\n", pc)
			os.Exit(1)
		}
		if c.PC == pc {
			if last == pc {
				same++
			}
			last = pc
			if same > 2 || true {
				if pc == 0x3469 {
					fmt.Printf("SUCCESS at %04x after %d steps, %d cycles, illegal=%d\n", pc, i, c.Cycles, c.Illegal)
					return
				}
				fmt.Printf("TRAP at %04x (test %d) A=%02x X=%02x Y=%02x P=%02x SP=%02x\n", pc, m[0x200], c.A, c.X, c.Y, c.P, c.SP)
				os.Exit(1)
			}
		}
	}
	fmt.Println("timeout")
}
