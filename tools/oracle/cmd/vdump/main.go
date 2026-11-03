package main

import (
	"fmt"
	"os"
	"strconv"

	"burgertime/tools/oracle/machine"
)

func main() {
	rs, _ := machine.LoadROMs(os.Args[1])
	frames, _ := strconv.Atoi(os.Args[2])
	m, _ := machine.New(rs)
	if v := os.Getenv("DSW1"); v != "" {
		x, _ := strconv.ParseInt(v, 0, 32)
		m.DSW1 = byte(x)
	}
	m.CPU.Reset()
	for f := 0; f < frames; f++ {
		m.RunCycles(machine.FrameCycles)
	}
	h := map[byte]int{}
	for _, b := range m.VRAM {
		h[b]++
	}
	fmt.Println("VRAM histogram:", h)
	h = map[byte]int{}
	for _, b := range m.CRAM {
		h[b]++
	}
	fmt.Println("CRAM histogram:", h)
	fmt.Printf("PC=%04x SP=%02x frame=%d\n", m.CPU.PC, m.CPU.SP, m.Frame)
	fmt.Println("zero page:")
	for i := 0; i < 256; i += 16 {
		fmt.Printf("%02x: % x\n", i, m.RAM[i:i+16])
	}
}
