package game

// PortTable lists the ported routines that original code may call, keyed by the
// address of the original routine. It exists for the development tools, which run
// partly ported games next to the reference CPU.
func PortTable() map[uint16]func(g *Game, r *Regs) {
	return map[uint16]func(g *Game, r *Regs){
		0xd36c: func(g *Game, r *Regs) { g.sub_d36c(r.X) },
		0xd4c6: func(g *Game, r *Regs) { q := g.sub_d4c6(r.X); r.A, r.X, r.Y, r.P = q.a, q.x, q.y, q.flags() },
		0xd148: func(g *Game, r *Regs) { g.sub_d148() },
		0xecf9: func(g *Game, r *Regs) { g.sub_ecf9() },
		0xd171: func(g *Game, r *Regs) { g.sub_d171() },
		0xd214: func(g *Game, r *Regs) { g.sub_d214() },
		0xd242: func(g *Game, r *Regs) { g.sub_d242() },
		0xd271: func(g *Game, r *Regs) { g.sub_d271() },
		0xece3: func(g *Game, r *Regs) { g.sub_ece3() },
		0xed63: func(g *Game, r *Regs) { g.sub_ed63() },
		0xeb60: func(g *Game, r *Regs) { r.A = g.sub_eb61(r.A); r.P = nz(r.X) },
		0xeb61: func(g *Game, r *Regs) { r.A = g.sub_eb61(r.A); r.P = nz(r.X) },
		0xfaa0: func(g *Game, r *Regs) { r.A = g.sub_faa0(r.A); r.P = nz(r.X) },
		0xd27e: func(g *Game, r *Regs) { g.sub_d27e() },
		0xd2b0: func(g *Game, r *Regs) { g.sub_d2b0() },
		0xd339: func(g *Game, r *Regs) { g.sub_d339() },
		0xd99b: func(g *Game, r *Regs) { g.sub_d99b() },
		0xdb6d: func(g *Game, r *Regs) { g.sub_db6d(r.Y) },
		0xdb5b: func(g *Game, r *Regs) { x, z := g.sub_db5b(r.Y); r.X = x; r.A = 0xff; r.P = 0; if z { r.P = FlagZ } },
		0xd899: func(g *Game, r *Regs) { g.sub_d899() },
		0xe775: func(g *Game, r *Regs) { g.sub_e775(); r.X = 6 },
		0xcb6e: func(g *Game, r *Regs) { r.A = r.Y; g.sub_cb6e(); r.X = g.RAM[zPlayer]; r.P = nz(r.A) },
		0xdb8c: func(g *Game, r *Regs) { g.sub_db8c() },
		0xdc2b: func(g *Game, r *Regs) { g.sub_dc2b() },
		0xdcbe: func(g *Game, r *Regs) { g.sub_dcbe() },
		0xdd7a: func(g *Game, r *Regs) { g.sub_dd7a() },
		0xde68: func(g *Game, r *Regs) { g.sub_de68() },
		0xddee: func(g *Game, r *Regs) { g.sub_ddee() },
		0xebad: func(g *Game, r *Regs) { g.sub_ebad() },
		0xe7e2: func(g *Game, r *Regs) { g.sub_e7e2() },
		0xe244: func(g *Game, r *Regs) { g.sub_e244() },
		0xe293: func(g *Game, r *Regs) { g.sub_e293() },
		0xe938: func(g *Game, r *Regs) { g.sub_e938() },
		0xe863: func(g *Game, r *Regs) { g.sub_e863() },
		0xe96f: func(g *Game, r *Regs) { g.sub_e96f() },
		0xea8f: func(g *Game, r *Regs) { g.sub_ea8f(r.A) },
		0xe6fd: func(g *Game, r *Regs) { g.sub_e6fd(r.Y) },
		0xe8cb: func(g *Game, r *Regs) { r.Y = g.sub_e8cb() },
		0xe1fe: func(g *Game, r *Regs) { g.sub_e1fe(r.X, r.Y) },
		0xe9f5: func(g *Game, r *Regs) { g.sub_e9f5() },
		0xcaff: func(g *Game, r *Regs) { g.waitFrames(r.X); r.X = 0 },
		0xed71: func(g *Game, r *Regs) { g.pauseWait(r.X); r.X = 0 },
		0xc362: func(g *Game, r *Regs) { g.sub_c362() },
		0xc34d: func(g *Game, r *Regs) { g.sub_c34d() },
		0xc333: func(g *Game, r *Regs) { g.sub_c333() },
		0xc976: func(g *Game, r *Regs) { g.sub_c976() },
		0xccbd: func(g *Game, r *Regs) { g.sub_ccbd() },
		0xc51e: func(g *Game, r *Regs) { g.sub_c51e(r.A) },
		0xca8f: func(g *Game, r *Regs) { g.printText(uint16(r.X) | uint16(r.Y)<<8) },
		0xca21: func(g *Game, r *Regs) { g.drawScore(r.X) },
		0xca2d: func(g *Game, r *Regs) { g.drawScoreAt(uint16(r.Y), r.X) },
		0xcce9: func(g *Game, r *Regs) { g.sub_cce9() },
		0xc282: func(g *Game, r *Regs) { g.sub_c282() },
		0xc29d: func(g *Game, r *Regs) { g.sub_c29d() },
		0xc2b1: func(g *Game, r *Regs) { g.sub_c2b1() },
		0xc2c5: func(g *Game, r *Regs) { g.sub_c2c5() },
		0xc2dd: func(g *Game, r *Regs) { g.sub_c2dd() },
		0xc2f5: func(g *Game, r *Regs) { g.sub_c2f5() },
		0xcd3e: func(g *Game, r *Regs) { g.drawBlockList() },
		0xcd73: func(g *Game, r *Regs) { g.setPtr(5, g.drawBlock(g.RAM[3], g.RAM[4], g.RAM[5], false)) },
		0xcd7c: func(g *Game, r *Regs) { g.setPtr(5, g.drawBlockAuto(g.RAM[3], g.RAM[4], g.RAM[5])) },
	}
}

// nz returns the N and Z status bits for a loaded value.
func nz(v byte) byte {
	var p byte
	if v == 0 {
		p |= FlagZ
	}
	p |= v & FlagN
	return p
}
