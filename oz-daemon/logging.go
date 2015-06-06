package daemon

import (
	"github.com/op/go-logging"
	"github.com/subgraph/oz/ipc"
	"log"
	"os"
)

func (d *daemonState) Debug(format string, args ...interface{}) {
	d.log.Debug(format, args...)
}
func (d *daemonState) Info(format string, args ...interface{}) {
	d.log.Info(format, args...)
}
func (d *daemonState) Notice(format string, args ...interface{}) {
	d.log.Notice(format, args...)
}
func (d *daemonState) Warning(format string, args ...interface{}) {
	d.log.Warning(format, args...)
}
func (d *daemonState) Error(format string, args ...interface{}) {
	d.log.Error(format, args...)
}
func (d *daemonState) Critical(format string, args ...interface{}) {
	d.log.Critical(format, args...)
}

func (d *daemonState) initializeLogging() {
	d.log = logging.MustGetLogger("oz")
	be := logging.NewChannelMemoryBackend(100)
	fbe := logging.NewBackendFormatter(be, format)
	d.memBackend = be
	stderr := logging.NewLogBackend(os.Stderr, "", log.LstdFlags)
	d.backends = []logging.Backend{
		stderr,
		fbe,
	}
	d.installBackends()
}

var format = logging.MustStringFormatter(
	"%{color}%{time:15:04:05} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}",
)

func (d *daemonState) addBackend(be logging.Backend) {
	d.backends = append(d.backends, be)
	d.installBackends()
}

func (d *daemonState) removeBackend(be logging.Backend) {
	newBackends := []logging.Backend{}
	for _, b := range d.backends {
		if b != be {
			newBackends = append(newBackends, b)
		}
	}
	d.backends = newBackends
	d.installBackends()
}

func (d *daemonState) installBackends() {
	if len(d.backends) == 1 {
		d.log.SetBackend(logging.AddModuleLevel(d.backends[0]))
		return
	}
	d.log.SetBackend(logging.MultiLogger(d.backends...))
}

type logFollower struct {
	daemon  *daemonState
	wrapper logging.Backend
	m       *ipc.Message
}

func (lf *logFollower) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	s := rec.Formatted(calldepth)
	if err := lf.m.Respond(&LogData{[]string{s}}); err != nil {
		lf.remove()
	}
	return nil
}

func (lf *logFollower) remove() {
	lf.daemon.removeBackend(lf.wrapper)
}

func (d *daemonState) followLogs(m *ipc.Message) {
	be := &logFollower{m: m, daemon: d}
	be.wrapper = logging.NewBackendFormatter(be, format)
	d.addBackend(be.wrapper)
}
