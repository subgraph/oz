package asm

import "syscall"

type instructionDescription struct {
	mnemonic        string
	realInstruction uint16
	takesJumps      bool
	takesK          bool
}

var instructionsByName = make(map[string]instructionDescription)
var instructionsByCode = make(map[uint16]instructionDescription)

func register(name string, real uint16, takesJ bool, takesK bool) {
	desc := instructionDescription{
		mnemonic:        name,
		realInstruction: real,
		takesJumps:      takesJ,
		takesK:          takesK,
	}
	instructionsByName[name] = desc
	instructionsByCode[real] = desc
}

// BPF_MOD is BPF_MOD - it is supported in Linux from v3.7+, but not in go's syscall...
const BPF_MOD = 0x90

// BPF_XOR is BPF_XOR - it is supported in Linux from v3.7+, but not in go's syscall...
const BPF_XOR = 0xa0

func init() {
	register("ret_k", syscall.BPF_RET|syscall.BPF_K, false, true)
	register("ret_x", syscall.BPF_RET|syscall.BPF_X, false, false)

	register("ld_abs", syscall.BPF_LD|syscall.BPF_W|syscall.BPF_ABS, false, true)
	register("ld_ind", syscall.BPF_LD|syscall.BPF_W|syscall.BPF_IND, false, true)
	register("ld_len", syscall.BPF_LD|syscall.BPF_W|syscall.BPF_LEN, false, false)
	register("ld_imm", syscall.BPF_LD|syscall.BPF_W|syscall.BPF_IMM, false, true)
	register("ld_mem", syscall.BPF_LD|syscall.BPF_W|syscall.BPF_MEM, false, true)
	register("ldx_len", syscall.BPF_LDX|syscall.BPF_W|syscall.BPF_LEN, false, false)
	register("ldx_imm", syscall.BPF_LDX|syscall.BPF_W|syscall.BPF_IMM, false, true)
	register("ldx_mem", syscall.BPF_LDX|syscall.BPF_W|syscall.BPF_MEM, false, true)

	register("st", syscall.BPF_ST, false, true)
	register("stx", syscall.BPF_STX, false, true)

	register("add_k", syscall.BPF_ALU|syscall.BPF_ADD|syscall.BPF_K, false, true)
	register("sub_k", syscall.BPF_ALU|syscall.BPF_SUB|syscall.BPF_K, false, true)
	register("mul_k", syscall.BPF_ALU|syscall.BPF_MUL|syscall.BPF_K, false, true)
	register("div_k", syscall.BPF_ALU|syscall.BPF_DIV|syscall.BPF_K, false, true)
	register("and_k", syscall.BPF_ALU|syscall.BPF_AND|syscall.BPF_K, false, true)
	register("or_k", syscall.BPF_ALU|syscall.BPF_OR|syscall.BPF_K, false, true)
	register("xor_k", syscall.BPF_ALU|BPF_XOR|syscall.BPF_K, false, true)
	register("lsh_k", syscall.BPF_ALU|syscall.BPF_LSH|syscall.BPF_K, false, true)
	register("rsh_k", syscall.BPF_ALU|syscall.BPF_RSH|syscall.BPF_K, false, true)
	register("mod_k", syscall.BPF_ALU|BPF_MOD|syscall.BPF_K, false, true)

	register("add_x", syscall.BPF_ALU|syscall.BPF_ADD|syscall.BPF_X, false, false)
	register("sub_x", syscall.BPF_ALU|syscall.BPF_SUB|syscall.BPF_X, false, false)
	register("mul_x", syscall.BPF_ALU|syscall.BPF_MUL|syscall.BPF_X, false, false)
	register("div_x", syscall.BPF_ALU|syscall.BPF_DIV|syscall.BPF_X, false, false)
	register("and_x", syscall.BPF_ALU|syscall.BPF_AND|syscall.BPF_X, false, false)
	register("or_x", syscall.BPF_ALU|syscall.BPF_OR|syscall.BPF_X, false, false)
	register("xor_x", syscall.BPF_ALU|BPF_XOR|syscall.BPF_X, false, false)
	register("lsh_x", syscall.BPF_ALU|syscall.BPF_LSH|syscall.BPF_X, false, false)
	register("rsh_x", syscall.BPF_ALU|syscall.BPF_RSH|syscall.BPF_X, false, false)
	register("mod_x", syscall.BPF_ALU|BPF_MOD|syscall.BPF_X, false, false)

	register("neg", syscall.BPF_ALU|syscall.BPF_NEG, false, false)

	register("tax", syscall.BPF_MISC|syscall.BPF_TAX, false, false)
	register("txa", syscall.BPF_MISC|syscall.BPF_TXA, false, false)

	register("jmp", syscall.BPF_JMP|syscall.BPF_JA, false, true)
	register("jgt_k", syscall.BPF_JMP|syscall.BPF_JGT|syscall.BPF_K, true, true)
	register("jge_k", syscall.BPF_JMP|syscall.BPF_JGE|syscall.BPF_K, true, true)
	register("jeq_k", syscall.BPF_JMP|syscall.BPF_JEQ|syscall.BPF_K, true, true)
	register("jset_k", syscall.BPF_JMP|syscall.BPF_JSET|syscall.BPF_K, true, true)

	register("jgt_x", syscall.BPF_JMP|syscall.BPF_JGT|syscall.BPF_X, true, false)
	register("jge_x", syscall.BPF_JMP|syscall.BPF_JGE|syscall.BPF_X, true, false)
	register("jeq_x", syscall.BPF_JMP|syscall.BPF_JEQ|syscall.BPF_X, true, false)
	register("jset_x", syscall.BPF_JMP|syscall.BPF_JSET|syscall.BPF_X, true, false)
}
