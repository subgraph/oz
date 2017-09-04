package asm

import (
	"log"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// The ASM format for BPF will on purpose be extremely simple - space separated values, all values in hex, no 0x
// prefixes. We will use mnemonics for the instructions.
// The first iteration will only read jump arguments if it's a jump instruction. It will also only
// read K if the instruction takes a K. The map below details the instructions:

// Add support for reading kill, trace etc?

func parseLine(s string) (unix.SockFilter, bool) {
	pieces := strings.Split(strings.Replace(strings.TrimSpace(s), "\t", " ", -1), " ")
	if len(pieces) == 0 || len(pieces) == 1 && pieces[0] == "" {
		return unix.SockFilter{}, false
	}

	inst, ok := instructionsByName[pieces[0]]
	if !ok {
		log.Printf("No instruction with name: %s known", pieces[0])
		return unix.SockFilter{}, false
	}

	index := 1
	filter := unix.SockFilter{}
	filter.Code = inst.realInstruction
	if inst.takesJumps {
		if len(pieces) < index+2 {
			log.Printf("Instruction %s requires jumps, but not enough given", pieces[0])
			return unix.SockFilter{}, false
		}
		tmp, err := strconv.ParseUint(pieces[index], 16, 8)
		if err != nil {
			log.Printf("Instruction %s has invalid jump: %s - %s", pieces[index], err.Error())
			return unix.SockFilter{}, false
		}
		filter.Jt = uint8(tmp)
		index++
		tmp, err = strconv.ParseUint(pieces[index], 16, 8)
		if err != nil {
			log.Printf("Instruction %s has invalid jump: %s - %s", pieces[index], err.Error())
			return unix.SockFilter{}, false
		}
		filter.Jf = uint8(tmp)
		index++
	}
	if inst.takesK {
		if len(pieces) < index+1 {
			log.Printf("Instruction %s takes K, but none given", pieces[0])
			return unix.SockFilter{}, false
		}

		tmp, err := strconv.ParseUint(pieces[index], 16, 32)
		if err != nil {
			log.Printf("Instruction %s has invalid K: %s - %s", pieces[index], err.Error())
			return unix.SockFilter{}, false
		}
		filter.K = uint32(tmp)
		index++
	}
	if len(pieces) > index {
		log.Printf("Instruction %s has extra values given: %v", pieces[index:])
		return unix.SockFilter{}, false
	}

	return filter, true
}

func parseLines(s []string) []unix.SockFilter {
	result := []unix.SockFilter{}
	for _, ss := range s {
		if r, ok := parseLine(ss); ok {
			result = append(result, r)
		}
	}
	return result
}

// Parse takes a string that contains a sock filter assembly program and returns the parsed representation
func Parse(s string) []unix.SockFilter {
	return parseLines(strings.Split(s, "\n"))
}
