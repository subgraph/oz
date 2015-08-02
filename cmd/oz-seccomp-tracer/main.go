package main

import (
	"runtime"
	ozseccomp "github.com/subgraph/oz/oz-seccomp"
)

func init() { runtime.LockOSThread() }

func main() {
	ozseccomp.Tracer()
}
