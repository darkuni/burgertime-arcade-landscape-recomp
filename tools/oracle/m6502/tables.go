package m6502

var (
	modeTab      [256]byte
	cycTab       [256]byte
	pcPenalty    [256]bool
	illegalHalts [256]bool
	undocNOP     [256]bool
)

func init() {
	for op := 0; op < 256; op++ {
		info, ok := opTab[byte(op)]
		if !ok {
			// unofficial opcode: treat as 1-byte NOP, flagged via Illegal count
			modeTab[op] = mImp
			cycTab[op] = 2
			switch op {
			case 0x04, 0x44, 0x64:
				modeTab[op], cycTab[op], undocNOP[op] = mZp, 3, true
			case 0x14, 0x34, 0x54, 0x74, 0xd4, 0xf4:
				modeTab[op], cycTab[op], undocNOP[op] = mZpx, 4, true
			case 0x0c:
				modeTab[op], cycTab[op], undocNOP[op] = mAbs, 4, true
			case 0x1c, 0x3c, 0x5c, 0x7c, 0xdc, 0xfc:
				modeTab[op], cycTab[op], undocNOP[op] = mAbx, 4, true
				pcPenalty[op] = true
			case 0x80, 0x82, 0x89, 0xc2, 0xe2:
				modeTab[op], cycTab[op], undocNOP[op] = mImm, 2, true
			case 0x1a, 0x3a, 0x5a, 0x7a, 0xda, 0xfa:
				undocNOP[op] = true
			}
			// KIL opcodes halt the CPU
			switch op {
			case 0x02, 0x12, 0x22, 0x32, 0x42, 0x52, 0x62, 0x72, 0x92, 0xb2, 0xd2, 0xf2:
				illegalHalts[op] = true
			}
			continue
		}
		modeTab[op] = info.mode
		n := info.name
		rmw := n == "ASL" || n == "LSR" || n == "ROL" || n == "ROR" || n == "INC" || n == "DEC"
		store := n == "STA" || n == "STX" || n == "STY"
		var c byte
		switch info.mode {
		case mImp, mAcc:
			c = 2
		case mImm:
			c = 2
		case mZp:
			c = 3
		case mZpx, mZpy:
			c = 4
		case mAbs:
			c = 4
		case mAbx, mAby:
			c = 4
		case mIzx:
			c = 6
		case mIzy:
			c = 5
		case mInd:
			c = 5
		case mRel:
			c = 2
		}
		if rmw {
			switch info.mode {
			case mZp:
				c = 5
			case mZpx, mAbs:
				c = 6
			case mAbx:
				c = 7
			}
		} else if store {
			switch info.mode {
			case mAbx, mAby:
				c = 5
			case mIzy:
				c = 6
			}
		} else if info.mode == mAbx || info.mode == mAby || info.mode == mIzy {
			pcPenalty[op] = true
		}
		switch n {
		case "JMP":
			if info.mode == mAbs {
				c = 3
			}
		case "JSR", "RTS", "RTI":
			c = 6
		case "BRK":
			c = 7
		case "PHA", "PHP":
			c = 3
		case "PLA", "PLP":
			c = 4
		}
		cycTab[op] = c
	}
}
