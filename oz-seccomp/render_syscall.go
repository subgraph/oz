package seccomp

import (
	"syscall"
)

type RenderingFunctions map[int]func(int, RegisterArgs) (string, error) 

func getRenderingFunctions() RenderingFunctions {
	r := map[int]func(pid int, args RegisterArgs) (string, error) { 
		syscall.SYS_ACCESS : render_access,
	}
	return r
}
