package main

import (
	ozseccomp "github.com/subgraph/oz/oz-seccomp"
	"runtime"
)

func init() { runtime.LockOSThread() }

func main() {
	ozseccomp.Tracer()
}
