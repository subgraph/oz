package compiler

import "syscall"

// These are only available from Linux 3.7+ and Go syscall doesn't have definitions for them
// The definitions are stable and doesn't vary between kernels.

const BPF_MOD = 0x90
const BPF_XOR = 0xA0

const OP_LOAD_VAL = syscall.BPF_LD | syscall.BPF_IMM
const OP_LOAD = syscall.BPF_LD | syscall.BPF_W | syscall.BPF_ABS
const OP_LOAD_MEM = syscall.BPF_LD | syscall.BPF_MEM
const OP_LOAD_MEM_X = syscall.BPF_LDX | syscall.BPF_MEM

const OP_STORE = syscall.BPF_ST
const OP_STORE_X = syscall.BPF_STX

const OP_ADD_X = syscall.BPF_ALU | syscall.BPF_ADD | syscall.BPF_X
const OP_SUB_X = syscall.BPF_ALU | syscall.BPF_SUB | syscall.BPF_X
const OP_MUL_X = syscall.BPF_ALU | syscall.BPF_MUL | syscall.BPF_X
const OP_DIV_X = syscall.BPF_ALU | syscall.BPF_DIV | syscall.BPF_X
const OP_MOD_X = syscall.BPF_ALU | BPF_MOD | syscall.BPF_X

const OP_AND_X = syscall.BPF_ALU | syscall.BPF_AND | syscall.BPF_X
const OP_OR_X = syscall.BPF_ALU | syscall.BPF_OR | syscall.BPF_X
const OP_XOR_X = syscall.BPF_ALU | BPF_XOR | syscall.BPF_X

const OP_LSH_X = syscall.BPF_ALU | syscall.BPF_LSH | syscall.BPF_X
const OP_RSH_X = syscall.BPF_ALU | syscall.BPF_RSH | syscall.BPF_X

const OP_JEQ_K = syscall.BPF_JMP | syscall.BPF_JEQ | syscall.BPF_K
const OP_JSET_K = syscall.BPF_JMP | syscall.BPF_JSET | syscall.BPF_K

const OP_JEQ_X = syscall.BPF_JMP | syscall.BPF_JEQ | syscall.BPF_X
const OP_JGT_X = syscall.BPF_JMP | syscall.BPF_JGT | syscall.BPF_X
const OP_JGE_X = syscall.BPF_JMP | syscall.BPF_JGE | syscall.BPF_X

const OP_JMP_K = syscall.BPF_JMP | syscall.BPF_JA

const OP_RET_K = syscall.BPF_RET | syscall.BPF_K
