package asm

import (
	"fmt"
	"log"
	"strings"

	"golang.org/x/sys/unix"
)

func dump(filter unix.SockFilter) (string, bool) {
	inst, ok := instructionsByCode[filter.Code]
	if !ok {
		log.Printf("Unknown instruction %x", filter.Code)
		return "", false
	}

	res := []string{inst.mnemonic}
	if inst.takesJumps {
		res = append(res, fmt.Sprintf("%02X", filter.Jt), fmt.Sprintf("%02X", filter.Jf))
	}
	if inst.takesK {
		res = append(res, fmt.Sprintf("%X", filter.K))
	}
	return strings.Join(res, "\t"), true
}

// Dump takes a series of sock filters and returns an assembler string that represents the program
func Dump(ss []unix.SockFilter) string {
	result := []string{}

	for _, s := range ss {
		if r, ok := dump(s); ok {
			result = append(result, r)
		}
	}

	return strings.Join(result, "\n") + "\n"
}
