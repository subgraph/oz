package oz

import (
	"github.com/op/go-logging"
	"os"
	"os/signal"
	"syscall"
)

func ReapChildProcs(log *logging.Logger, callback func(int, syscall.WaitStatus)) chan os.Signal {
	sigs := make(chan os.Signal, 3)
	signal.Notify(sigs, syscall.SIGCHLD)
	go func() {
		for {
			<-sigs
			handleSIGCHLD(log, callback)
		}
	}()
	return sigs
}

func handleSIGCHLD(log *logging.Logger, callback func(int, syscall.WaitStatus)) {
	var wstatus syscall.WaitStatus
	for {
		pid, err := syscall.Wait4(-1, &wstatus, syscall.WNOHANG, nil)
		switch err {
		case syscall.ECHILD:
			return

		case syscall.EINTR:

		case nil:
			if pid == 0 {
				return
			}
			callback(pid, wstatus)

		default:
			if log != nil {
				log.Warning("syscall.Wait4() returned error: %v", err)
			}
			return
		}
	}
}
