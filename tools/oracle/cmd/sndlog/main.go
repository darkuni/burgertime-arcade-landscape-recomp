// sndlog runs the reference sound board on its own and prints every AY write.
//
//	sndlog <rom dir> <total sound cycles> <cmd@cycle>...
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"burgertime/tools/oracle/machine"
)

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
	// run the self test and start-up until the idle loop is reached
	line := 0
	lineCyc := 0
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
	s.Record = true
	s.Log = nil
	ei := 0
	for s.CPU.Cycles-base < total {
		for ei < len(evs) && s.CPU.Cycles-base >= evs[ei].at {
			s.Command(evs[ei].cmd)
			ei++
		}
		step()
	}
	for _, w := range s.Log {
		fmt.Printf("%d %d %02x %02x\n", w.Cycle-base, w.Chip, w.Reg, w.Val)
	}
}
