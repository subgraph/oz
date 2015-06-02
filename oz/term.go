package main
import (
	"syscall"
	"unsafe"
	"os"
	"os/signal"
	"fmt"
)

type winsize struct {
	Height uint16
	Width uint16
	x uint16
	y uint16
}
type State struct {
	termios Termios
}

const (
	getTermios = syscall.TCGETS
	setTermios = syscall.TCSETS
)

type Termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Cc     [20]byte
	Ispeed uint32
	Ospeed uint32
}

// MakeRaw put the terminal connected to the given file descriptor into raw
// mode and returns the previous state of the terminal so that it can be
// restored.
func MakeRaw(fd uintptr) (*State, error) {
	var oldState State
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, getTermios, uintptr(unsafe.Pointer(&oldState.termios))); err != 0 {
		return nil, err
	}

	newState := oldState.termios

	newState.Iflag &^= (syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON)
	newState.Oflag &^= syscall.OPOST
	newState.Lflag &^= (syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN)
	newState.Cflag &^= (syscall.CSIZE | syscall.PARENB)
	newState.Cflag |= syscall.CS8

	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, setTermios, uintptr(unsafe.Pointer(&newState))); err != 0 {
		return nil, err
	}
	return &oldState, nil
}

// Restore restores the terminal connected to the given file descriptor to a
// previous state.
func RestoreTerminal(fd uintptr, state *State) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(setTermios), uintptr(unsafe.Pointer(&state.termios)))
	return err
}

func SetRawTerminal(fd uintptr) (*State, error) {
	oldState, err := MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		_ = <-c
		RestoreTerminal(fd, oldState)
		os.Exit(0)
	}()
	return oldState, err
}

func GetWinsize(fd uintptr) (*winsize, syscall.Errno) {
	ws := &winsize{}
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(ws)))
	return ws, err
}

func SetWinsize(fd uintptr, ws *winsize) syscall.Errno {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
	return err
}

func HandleResize(fd int) {
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGWINCH)
	go func() {
		for {
			<-sigs
			handleSIGWINCH(fd)
		}
	}()
	handleSIGWINCH(fd)
}

func handleSIGWINCH(fd int) {
	ws,errno := GetWinsize(0)
	if errno != 0 {
		fmt.Printf("error reading winsize: %v\n", errno.Error())
		return
	}
	if errno := SetWinsize(uintptr(fd), ws); errno != 0 {
		fmt.Printf("error setting winsize: %v", errno.Error())
	}
}