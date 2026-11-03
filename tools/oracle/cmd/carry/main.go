package main

import (
	"fmt"
	"math/rand"

	"burgertime/tools/oracle/machine"
)

func main() {
	rs, _ := machine.LoadROMs("../rom")
	m, _ := machine.New(rs)
	m.NoSound = true
	m.CPU.Reset()
	rng := rand.New(rand.NewSource(3))
	cnt := map[int]int{}
	lastC := uint16(0)
	hist := map[uint16]int{}
	cw := map[byte]bool{}
	for _, o := range []byte{0x69, 0x65, 0x75, 0x6d, 0x7d, 0x79, 0x61, 0x71, 0xe9, 0xe5, 0xf5, 0xed, 0xfd, 0xf9, 0xe1, 0xf1, 0xc9, 0xc5, 0xd5, 0xcd, 0xdd, 0xd9, 0xc1, 0xd1, 0xe0, 0xe4, 0xec, 0xc0, 0xc4, 0xcc, 0x0a, 0x06, 0x16, 0x0e, 0x1e, 0x4a, 0x46, 0x56, 0x4e, 0x5e, 0x2a, 0x26, 0x36, 0x2e, 0x3e, 0x6a, 0x66, 0x76, 0x6e, 0x7e, 0x38, 0x18, 0x28, 0x40} {
		cw[o] = true
	}
	dir := byte(0xff)
	it := 0
	for it < 20000 {
		pc := m.CPU.PC
		if pc == 0xc0ac {
			hist[lastC]++
			cnt[int(m.CPU.P&1)]++
			if it%20 == 0 {
				_ = rng
			}
		}
		if pc == 0xc086 || pc == 0xcb05 {
			it++
			if it%20 == 0 {
				dir = 0xff
				if r := rng.Intn(7); r < 4 {
					dir &^= 1 << uint(r)
				}
				if rng.Intn(6) == 0 {
					dir &^= 0x10
				}
			}
			m.P1 = dir
			switch it {
			case 120, 900, 1500, 2600, 4000:
				m.System |= 0x40
				m.CPU.IRQ = true
			case 124, 904, 1504, 2604, 4004:
				m.System &^= 0x40
			case 300, 1700, 2800:
				m.System &^= 1
			case 304, 1704, 2804:
				m.System |= 3
			}
		}
		m.RunCycles(1)
		if cw[m.CPU.LastOp] {
			lastC = m.CPU.LastPC
		}
	}
	fmt.Println(cnt)
	for k, v := range hist {
		fmt.Printf("%04x %d\n", k, v)
	}
}
