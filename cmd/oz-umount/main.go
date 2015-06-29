package main

import (
	"runtime"

	"github.com/subgraph/oz/oz-mount"
)

func init() {
	runtime.LockOSThread()
	runtime.GOMAXPROCS(1)
}

func main() {
	defer runtime.UnlockOSThread()

	mount.Main(mount.UMOUNT)
}
