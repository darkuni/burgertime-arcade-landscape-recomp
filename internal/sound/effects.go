package sound

// Command handling (the program's interrupt routine at f255) and the sound
// effects. An effect is a straight program with timing loops and writes to the
// second chip; since nothing it does depends on the outside world, it is
// produced as a time-stamped script when the command arrives.

// effect priorities (table at f2c6): a running effect is only replaced by one
// whose priority is not lower.
var effectPrio = [15]byte{0, 3, 3, 2, 7, 4, 9, 9, 6, 0x0a, 8, 0x0c, 0x0b, 5, 1}

// Command delivers a command byte from the main board at the current time.
func (s *Seq) Command(v byte) {
	z := &s.z
	z[zCmd] = v
	c := 10 + 13 + 4 + 3 + 2
	if v&0xf0 == 0 {
		s.musicCommand(v, c+3)
		return
	}
	c += 2
	if v&0xf0 != 0x10 {
		s.busy += c + 3 + 22
		return
	}
	x := v & 0x0f
	c += 3 + 2 + 2
	if x == 0 || x >= 0x0f {
		s.busy += c + 3 + 22
		return
	}
	c += 2 + 3 + 4
	if z[zPrio] > effectPrio[x] {
		s.busy += c + 2 + 3 + 22
		return
	}
	c += 3 + 4 + 3 + 2 + 2 + 2 + 4 + 3 + 4 + 3 + 2 + 2 + 2 + 5
	z[zPrio] = effectPrio[x]
	s.busy += c
	s.startEffect(x)
}

// musicCommand is the handler at f297 for commands 0..5.
func (s *Seq) musicCommand(v byte, c int) {
	z := &s.z
	z[zSaved] = z[zEnable]
	z[zEnable] = 0
	z[zVol], z[zVol+1], z[zVol+2] = 0x0f, 0x0e, 0x0e
	c += 3 + 3 + 2 + 3 + 2 + 3 + 2 + 3 + 2 + 3 + 3 + 2 + 2
	x := int(v) * 2
	if x >= 0x0c {
		z[zEnable] = z[zSaved]
		s.busy += c + 3 + 3 + 3 + 3 + 22
		return
	}
	c += 2 + 4 + 3 + 4 + 3
	// f306: mixer of the first chip
	s.emit(s.Now+uint64(c+2+4+2), 0, 7, 0xf8)
	c += 2 + 4 + 2 + 4 + 6 + 5
	switch v {
	case 0:
		s.reinit()
		s.busy += c + 200
	case 1, 2:
		base := uint16(0xf3a4)
		if v == 2 {
			base = 0xf3c2
		}
		c += s.f3c8(c)
		for i := 0; i < 6; i++ {
			z[zPtr+i] = s.rb(base + uint16(i))
		}
		c += 6*(4+4+2+2) + 2 + 3 + 3 + 3 + 3
		z[zEnable] = 0x0f
		z[zBusyTune] = 0x0f
		s.busy += c + 22
	case 3, 4, 5:
		c += 2 + 3 + 3 + 3
		if v == 5 {
			z[zTune] = 0xff
		} else {
			z[zTune] = 0
		}
		if z[zBusyTune] != 0 {
			z[zEnable] = z[zSaved]
			s.busy += c + 3 + 3 + 3 + 22
			return
		}
		base := uint16(0xf380)
		if v == 5 {
			base = 0xf386
		}
		for i := 0; i < 6; i++ {
			z[zPtr+i] = s.rb(base + uint16(i))
		}
		c += 2 + 6*(4+4+2+2)
		z[zVol] = 0x0c
		z[zVol+1] = 0
		if v == 5 {
			z[zVol+1] = 0x0c
		}
		z[zVol+2] = 0x0b
		z[zEnable] = 0x0b
		c += 2 + 3 + 2 + 3 + 2 + 3 + 3 + 6
		// f311: the three volumes
		for i := 0; i < 3; i++ {
			s.emit(s.Now+uint64(c+2+4+3), 0, 8+byte(i), z[zVol+i])
			c += 2 + 4 + 3 + 4
		}
		s.busy += c + 6 + 3 + 22
	}
}

// f3c8: silence both chips, restore the mixer and clear the music state.
func (s *Seq) f3c8(c int) int {
	z := &s.z
	for r := 0; r < 16; r++ {
		v := s.rb(0xf064 + uint16(r))
		w := s.Now + uint64(c+r*30+11)
		s.emit(w, 0, byte(r), v)
		s.emit(w, 1, byte(r), v)
	}
	t := 6 + 7 + 16*30 + 6
	s.emit(s.Now+uint64(c+t+2+4+2), 0, 7, 0xf8)
	t += 2 + 4 + 2 + 4 + 6 + 6
	z[zPhase] = 0
	for i := 0; i < 0x0b; i++ {
		z[zIdx+i] = 0
	}
	return t + 11*(4+2+2) + 6
}

// ---- effect scripts ------------------------------------------------------

// vm builds the script of an effect: c is the effect-local cycle counter.
type vm struct {
	s  *Seq
	c  int
	ev []effEvent
}

func (v *vm) w1(reg, val byte) {
	v.ev = append(v.ev, effEvent{at: v.c, chip: 1, reg: reg & 15, val: val})
}

// page-cross penalty of an indexed read
func cross(base uint16, idx byte) int {
	if (base&0xff)+uint16(idx) > 0xff {
		return 1
	}
	return 0
}

// startEffect replaces the running effect with the one for command x (1..14).
func (s *Seq) startEffect(x byte) {
	v := &vm{s: s}
	switch x {
	case 1:
		v.tableEffect(4)
	case 2:
		v.stepEffect(0x24, 0xd2, 0xf5, 0x20, 0, 8)
	case 3:
		v.stepEffect(0x10, 0x07, 0xf6, 0x0c, 2, 9)
	case 4:
		v.tableEffect(5)
	case 5:
		v.effect449()
	case 6:
		v.effect4ae()
	case 7:
		v.tableEffect(1)
	case 8:
		v.effect6d8()
	case 9:
		v.tableEffect(0)
	case 10:
		v.tableEffect(2)
	case 11:
		v.effect560()
	case 12:
		v.tableEffect(6)
	case 13:
		v.effect4ef()
	case 14:
		v.tableEffect(3)
	}
	s.eff, s.effPos, s.ec = v.ev, 0, 0
}

// finish is the end of every effect: release the priority, silence the chip (f023).
func (v *vm) finish() {
	v.c += 2 + 3 // LDA #0, STA $3e
	v.ev = append(v.ev, effEvent{at: v.c, prio: true})
	v.c += 3 + 2 + 6 // JMP, CLI, JSR
	v.f04a()
}

// f04a: the second chip gets its 11 start values.
func (v *vm) f04a() {
	v.c += 2 + 3 + 2
	for r := 0; r < 11; r++ {
		v.c += 3 + 4 + 4
		v.w1(byte(r), v.s.rb(0xf064+uint16(r)))
		v.c += 4 + 2 + 5 + 3 + 2
		if r < 10 {
			v.c += 3
		} else {
			v.c += 2
		}
	}
	v.c += 6
}

// f768: clears the effect variables and silences the chip.
func (v *vm) f768() {
	v.c += 6 + 4
	for x := 0x10; x >= 0; x-- {
		v.s.z[0x21+x] = 0
		v.c += 4 + 2
		if x > 0 {
			v.c += 3
		} else {
			v.c += 2
		}
	}
	v.c += 6
	v.f04a()
	v.c += 6
}

// f753: writes register / value pairs from the table at f71d, from y up to end.
func (v *vm) f753(y, end byte) {
	v.c += 3
	for {
		v.c += 4 + 4 + 2 + 4
		reg := v.s.rb(0xf71d + uint16(y))
		val := v.s.rb(0xf71d + uint16(y) + 1)
		v.w1(reg, val)
		v.c += 4 + 2 + 3
		y += 2
		if y == end {
			v.c += 2
			break
		}
		v.c += 3
	}
	v.c += 6
}

// f775: writes the six tone registers from $29..$2e.
func (v *vm) f775() {
	z := &v.s.z
	v.c += 2
	for x := 5; x >= 0; x-- {
		v.c += 4 + 4
		v.w1(byte(x), z[0x29+x])
		v.c += 4 + 2
		if x > 0 {
			v.c += 3
		} else {
			v.c += 2
		}
	}
	v.c += 6
}

// f783: writes register $21 with the value $22.
func (v *vm) f783() {
	z := &v.s.z
	v.c += 3 + 4 + 3
	v.w1(z[0x21], z[0x22])
	v.c += 4 + 6
}

// f78e: the delay routine; y units of about 1300 cycles.
func (v *vm) f78e(y byte) {
	n := int(y)
	if n == 0 {
		n = 256
	}
	v.c += n*1310 + 5
}

// f5a4: fetches the next three bytes of the table at ($2f) into $21..$23.
func (v *vm) f5a4() {
	z := &v.s.z
	base := uint16(z[0x2f]) | uint16(z[0x30])<<8
	v.c += 3
	for x := z[0x25]; x != 3; x++ {
		v.c += 3 + 5 + cross(base, z[0x26]) + 4 + 5 + 2 + 2
		z[0x21+x] = v.s.rb(base + uint16(z[0x26]))
		z[0x26]++
		if x != 2 {
			v.c += 3
		} else {
			v.c += 2
		}
	}
	z[0x25] = 0
	v.c += 2 + 3 + 6
}

// f5b8: sets register $29 to $21 and register $2a to $22, then delays $23 units.
func (v *vm) f5b8() {
	z := &v.s.z
	v.c += 3 + 4 + 3
	v.w1(z[0x29], z[0x21])
	v.c += 4 + 3 + 4 + 3
	v.w1(z[0x2a], z[0x22])
	v.c += 4 + 3 + 6
	v.f78e(z[0x23])
	v.c += 6
}

// stepEffect: effects 2 and 3 (f3db, f412): a list of (tone, volume, length) steps.
func (v *vm) stepEffect(initEnd byte, tabLo, tabHi byte, initStart byte, r1, r2 byte) {
	z := &v.s.z
	v.f768()
	v.c += 2 + 2 + 6
	v.f753(initStart, initEnd)
	z[0x2f], z[0x30] = tabLo, tabHi
	z[0x25], z[0x29], z[0x2a] = 0, r1, r2
	v.c += 4*2 + 3*5 + 6
	for {
		v.c += 6
		v.f5a4()
		v.c += 3 + 2
		if z[0x22] == 0xff {
			v.c += 3
			break
		}
		v.c += 2 + 6
		v.f5b8()
		v.c += 3
	}
	v.c += 6
	v.f768()
	v.finish()
}

// effect449 (command 5): a falling tone on two voices.
func (v *vm) effect449() {
	z := &v.s.z
	v.f768()
	v.c += 2 + 2 + 6
	v.f753(0x24, 0x2a)
	z[0x27], z[0x22], z[0x2b], z[0x2d] = 0x10, 0x0f, 0x57, 0x74
	v.c += 4 * 5
	for {
		z[0x2b] -= 0x0f
		z[0x2d] -= 6
		v.c += 10 + 10
		v.c += 6
		v.f775()
		z[0x21] = 9
		v.c += 5 + 6
		v.f783()
		z[0x21]++
		v.c += 5 + 6
		v.f783()
		v.c += 2 + 6
		v.f78e(2)
		z[0x2b]++
		z[0x2d] += 3
		v.c += 5 + 15 + 6
		v.f775()
		v.c += 2 + 6
		v.f78e(1)
		v.c += 3 + 2 + 2
		if z[0x27]&1 != 0 {
			z[0x22]--
			v.c += 1 + 5
		}
		z[0x27]--
		v.c += 5 + 2
		if z[0x27] == 0 {
			break
		}
		v.c += 1 + 3
	}
	v.c += 6
	v.f768()
	v.finish()
}

// effect4ae (command 6): three falling blips.
func (v *vm) effect4ae() {
	z := &v.s.z
	v.f768()
	v.c += 2 + 2 + 6
	v.f753(0x10, 0x14)
	z[0x27], z[0x23], z[0x29], z[0x2a] = 3, 3, 2, 9
	v.c += 2 + 3 + 3 + 2 + 3 + 2 + 3
	for {
		z[0x22] = 7
		v.c += 2 + 3
		for {
			v.c += 6
			v.f5b8()
			z[0x22]++
			v.c += 5 + 3 + 2
			if z[0x22] == 0x0f {
				v.c += 2
				break
			}
			v.c += 3
		}
		z[0x22] = 0
		v.c += 2 + 3 + 6
		v.f5b8()
		v.c += 2 + 6
		v.f78e(0x0f)
		z[0x27]--
		v.c += 5
		if z[0x27] == 0 {
			v.c += 2
			break
		}
		v.c += 3
	}
	v.c += 6
	v.f768()
	v.finish()
}

// effect4ef (command 13): a descending run through the table at f538.
func (v *vm) effect4ef() {
	z := &v.s.z
	v.f768()
	v.c += 2 + 2 + 6
	v.f753(0x2a, 0x36)
	z[0x27] = 1
	v.c += 2 + 3
	for {
		z[0x22] = 0
		v.c += 2 + 3
		for {
			// f527
			v.c += 6 + 3 + 4 + 3 + 2 + 4 + 3 + 2 + 3 + 6
			x := z[0x22]
			z[0x2b] = v.s.rb(0xf538 + uint16(x))
			z[0x21] = v.s.rb(0xf538 + uint16(x) + 1)
			z[0x22] = x + 2
			z[0x2d] = z[0x2b] - 0x20
			v.c += 2 + 3 + 2 + 3
			v.c += 6
			v.f775()
			v.c += 3 + 6
			v.f78e(z[0x21])
			v.c += 3 + 2 + 2
			if z[0x2b] == 0 {
				v.c++
				break
			}
			v.c += 1
		}
		z[0x27]--
		v.c += 5
		if z[0x27] == 0 {
			v.c += 2
			break
		}
		v.c += 3
	}
	v.c += 6
	v.f768()
	v.finish()
}

// effect560 (command 11): two interleaved falling tones.
func (v *vm) effect560() {
	z := &v.s.z
	v.f768()
	v.c += 2 + 2 + 6
	v.f753(0, 0x0c)
	z[0x2b], z[0x2d] = 0xe0, 0x90
	v.c += 4*2 + 2*3
	for {
		z[0x2b] -= 4
		z[0x2d] -= 4
		v.c += 10 + 10 + 6
		v.f775()
		v.c += 2 + 6
		v.f78e(1)
		z[0x2b]++
		z[0x2d]++
		v.c += 10 + 6
		v.f775()
		v.c += 2 + 6
		v.f78e(1)
		v.c += 3 + 2 + 2
		if z[0x2b] == 0x64 {
			v.c++
			break
		}
		v.c++
	}
	v.c += 6
	v.f768()
	v.finish()
}

// effect6d8 (command 8): a rising warble.
func (v *vm) effect6d8() {
	z := &v.s.z
	v.f768()
	v.c += 2 + 2 + 6
	v.f753(0, 0x0c)
	z[0x2b], z[0x2d] = 0xe3, 0xa0
	v.c += 4*2 + 2*3
	for {
		z[0x2b] -= 4
		z[0x2d] -= 3
		v.c += 10 + 10 + 6
		v.f775()
		v.c += 2 + 6
		v.f78e(1)
		z[0x2b]++
		z[0x2d]++
		v.c += 2 + 10 + 6
		v.f775()
		v.c += 2 + 6
		v.f78e(1)
		v.c += 3 + 2 + 2
		if z[0x2b] < 0x30 {
			v.c++
			break
		}
		v.c += 1
	}
	v.c += 6
	v.f768()
	v.finish()
}

// tableEffect: the effects driven by the data tables at fa9a / faa8 (sets $44 = n).
func (v *vm) tableEffect(n byte) {
	z := &v.s.z
	s := v.s
	v.c += 2 + 3 + 3
	v.c += 6
	v.f768()
	z[0x44] = n
	x := uint16(n) * 2
	initP := uint16(s.rb(0xfaa8+x)) | uint16(s.rb(0xfaa9+x))<<8
	noteP := uint16(s.rb(0xfa9a+x)) | uint16(s.rb(0xfa9b+x))<<8
	v.c += 3 + 2 + 2 + 4*4 + 3*4 + 2
	// the register / value pairs, ended by a $80 where the next register would be
	y := byte(0)
	for {
		v.c += 5 + cross(initP, y) + 4
		reg := s.rb(initP + uint16(y))
		y++
		v.c += 2 + 5 + cross(initP, y)
		val := s.rb(initP + uint16(y))
		v.w1(reg, val)
		v.c += 4
		y++
		v.c += 2 + 5 + cross(initP, y) + 2
		if s.rb(initP+uint16(y)) == 0x80 {
			v.c += 2
			break
		}
		v.c += 3
	}
	y++
	z[0x26] = y
	delayPos := initP + uint16(y)
	v.c += 2 + 3 + 2
	y = 0
	for {
		v.c += 2 + 4 + 5 + cross(noteP, y)
		v.w1(0, s.rb(noteP+uint16(y)))
		// (the write goes to the register selected by the preceding $8000 store)
		y++
		v.c += 2 + 4 + 5 + cross(noteP, y)
		v.w1(2, s.rb(noteP+uint16(y)))
		v.c += 4 + 6
		v.delayF6c1(s.rb(delayPos))
		y++
		v.c += 2 + 5 + cross(noteP, y) + 2
		if s.rb(noteP+uint16(y)) == 0x81 {
			v.c += 2
			break
		}
		v.c += 3
	}
	v.c += 6
	v.f768()
	v.finish()
}

// delayF6c1: the delay of the table effects: n units of about 2000 cycles.
func (v *vm) delayF6c1(n byte) {
	v.c += 2 + 3 + 3 + 5 + 3 + 2 + 3 + 2015
	for left := n; ; {
		left--
		v.c += 5
		if left == 0 {
			v.c += 2
			break
		}
		v.c += 3 + 2047
	}
	v.c += 4 + 2 + 6
}
