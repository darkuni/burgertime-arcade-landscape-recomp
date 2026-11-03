// sndwav renders a sound command to a 16-bit mono WAV file (development tool).
package main

import (
	"encoding/binary"
	"os"
	"strconv"

	"burgertime/internal/assets"
	"burgertime/internal/sound"
)

func main() {
	r, err := assets.Load(os.Args[1])
	if err != nil {
		panic(err)
	}
	cmd, _ := strconv.ParseInt(os.Args[2], 0, 32)
	secs, _ := strconv.Atoi(os.Args[3])
	const rate = 44100
	s := sound.NewSynth(r.Sound[:], rate)
	s.Command(uint8(cmd))
	out := make([]float32, rate*secs)
	s.Render(out)
	f, _ := os.Create(os.Args[4])
	defer f.Close()
	n := uint32(len(out) * 2)
	hdr := []any{[]byte("RIFF"), 36 + n, []byte("WAVEfmt "), uint32(16), uint16(1), uint16(1), uint32(rate), uint32(rate * 2), uint16(2), uint16(16), []byte("data"), n}
	for _, h := range hdr {
		binary.Write(f, binary.LittleEndian, h)
	}
	for _, v := range out {
		binary.Write(f, binary.LittleEndian, int16(v*32767))
	}
}
