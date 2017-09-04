package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"

	"github.com/twtiger/gosecco"
	"github.com/twtiger/gosecco/asm"
)

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func checkArgs() bool {
	return len(os.Args) < 4 ||
		(os.Args[1] != "white" && os.Args[1] != "black") ||
		(os.Args[2] != "true" && os.Args[2] != "false") ||
		!fileExists(os.Args[3])
}

func main() {
	if checkArgs() {
		fmt.Println("Usage: tester [white|black] <enforce> <filename>")
		return
	}

	whiteList := os.Args[1] == "white"
	enforce := os.Args[2] == "true"
	filename := os.Args[3]

	gosecco.CheckSupport()
	var e error
	var filters []unix.SockFilter

	if whiteList {
		filters, e = gosecco.Compile(filename, enforce)
	} else {
		filters, e = gosecco.CompileBlacklist(filename, enforce)
	}

	if e != nil {
		fmt.Printf("Had error when compiling: %#v - %s\n", e, e.Error())
	} else {
		fmt.Print(asm.Dump(filters))
	}
}
