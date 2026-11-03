// shot boots the reference machine, applies a scripted sequence and writes a PNG.
package main

import (
	"fmt"
	"image/png"
	"os"
	"strconv"

	"burgertime/internal/assets"
	"burgertime/internal/video"
	"burgertime/tools/oracle/machine"
)

func main() {
	rom := os.Args[1]
	frames, _ := strconv.Atoi(os.Args[2])
	out := os.Args[3]
	coinAt, startAt := -1, -1
	if v := os.Getenv("COIN"); v != "" {
		coinAt, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("START"); v != "" {
		startAt, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("SMODE"); v != "" {
		fmt.Sscan(v, &machine.ScrambleMode)
	}
	if v := os.Getenv("ARMMASK"); v != "" {
		fmt.Sscan(v, &machine.ArmMask)
	}
	rs, _ := machine.LoadROMs(rom)
	m, err := machine.New(rs)
	if err != nil {
		panic(err)
	}
	if v := os.Getenv("DSW1"); v != "" {
		x, _ := strconv.ParseInt(v, 0, 32)
		m.DSW1 = byte(x)
	}
	if v := os.Getenv("DSW2"); v != "" {
		x, _ := strconv.ParseInt(v, 0, 32)
		m.DSW2 = byte(x)
	}
	m.CPU.Reset()
	for f := 0; f < frames; f++ {
		if f == coinAt {
			m.System |= 0x40
			m.CPU.IRQ = true
		}
		if f == coinAt+3 {
			m.System &^= 0x40
		}
		if f == startAt {
			m.System &^= 0x01
		}
		if f == startAt+4 {
			m.System |= 0x01
		}
		m.RunCycles(machine.FrameCycles)
	}
	r, err := assets.Load(rom)
	if err != nil {
		panic(err)
	}
	rend := video.New(r.Decode())
	var st video.State
	st.VRAM, st.CRAM, st.Palette = m.VRAM, m.CRAM, m.Palette
	st.Scroll[0], st.VidCtl = m.Scroll[0], m.VidCtl
	img := rend.Image(&st)
	f, _ := os.Create(out)
	png.Encode(f, img)
	f.Close()
	fmt.Println("wrote", out, "frame", m.Frame, "PC", fmt.Sprintf("%04x", m.CPU.PC), "illegal", m.CPU.Illegal)
}
