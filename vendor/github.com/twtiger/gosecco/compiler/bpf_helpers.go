package compiler

func bpfClass(code uint16) uint16 {
	return code & 0x07
}

func bpfSize(code uint16) uint16 {
	return code & 0x18
}

func bpfMode(code uint16) uint16 {
	return code & 0xe0
}

func bpfOp(code uint16) uint16 {
	return code & 0xf0
}

func bpfMiscOp(code uint16) uint16 {
	return code & 0xf8
}

func bpfSrc(code uint16) uint16 {
	return code & 0x08
}
