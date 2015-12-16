package seccomp

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/subgraph/oz"
	"github.com/subgraph/oz/fs"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path"
	"syscall"
)

// #include "sys/ptrace.h"
import "C"

const (
	STRINGARG = iota + 1
	PTRARG
	INTARG
)

type SystemCallArgs []int

func Tracer() {

	var train = false
	var cmd string
	var cmdArgs []string
	var p *oz.Profile

	var noprofile = flag.Bool("train", false, "Training mode")
	var debug = flag.Bool("debug", false, "Debug")
	var appendpolicy = flag.Bool("append", false, "Append to existing policy if exists")
	var trainingoutput = flag.String("output", "", "Training policy output file")

	flag.Parse()

	var args = flag.Args()

	if *noprofile == true {
		train = true
		cmd = "/usr/local/bin/oz-seccomp"
		cmdArgs = append([]string{"-mode=train"}, args...)
	} else {
		p = new(oz.Profile)
		if err := json.NewDecoder(os.Stdin).Decode(&p); err != nil {
			log.Error("unable to decode profile data: %v", err)
			os.Exit(1)
		}
		if p.Seccomp.Mode == oz.PROFILE_SECCOMP_TRAIN {
			train = true
		}
		*debug = p.Seccomp.Debug
		cmd = args[0]
		cmdArgs = args[1:]
	}

	var cpid = 0
	done := false

	log.Info("Tracer running command (%v) arguments (%v)\n", cmd, cmdArgs)
	c := exec.Command(cmd)
	c.SysProcAttr = &syscall.SysProcAttr{Ptrace: true}
	c.Env = os.Environ()
	c.Args = append(c.Args, cmdArgs...)

	if *noprofile == false {

		pi, err := c.StdinPipe()
		if err != nil {
			fmt.Errorf("error creating stdin pipe for tracer process: %v", err)
			os.Exit(1)
		}
		jdata, err := json.Marshal(p)
		if err != nil {
			fmt.Errorf("Unable to marshal seccomp state: %+v", err)
			os.Exit(1)
		}
		io.Copy(pi, bytes.NewBuffer(jdata))
		pi.Close()
	}
	children := make(map[int]bool)
	renderFunctions := getRenderingFunctions()

	trainingset := make(map[int]bool)
	trainingargs := make(map[int]map[int][]uint)

	if err := c.Start(); err == nil {
		cpid = c.Process.Pid
		children[c.Process.Pid] = true
		var s syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &s, syscall.WALL, nil)
		children[pid] = true
		if err != nil {
			log.Error("Error (wait4) err:%v pid:%i", err, pid)
		}
		log.Info("Tracing child pid: %v\n", pid)
		for done == false {
			pflags := unix.PTRACE_O_TRACESECCOMP
			pflags |= unix.PTRACE_O_TRACEFORK
			pflags |= unix.PTRACE_O_TRACEVFORK
			pflags |= unix.PTRACE_O_TRACECLONE
			pflags |= C.PTRACE_O_EXITKILL

			syscall.PtraceSetOptions(pid, pflags)
			syscall.PtraceCont(pid, 0)
			pid, err = syscall.Wait4(-1, &s, syscall.WALL, nil)
			if err != nil {
				log.Error("Error (wait4) err:%v pid:%i children:%v\n", err, pid, children)
				done = true
				continue
			}
			children[pid] = true
			if s.Exited() == true {
				delete(children, pid)
				log.Info("Child pid %v finished.\n", pid)
				if len(children) == 0 {
					done = true
				}
				continue
			}
			if s.Signaled() == true {
				log.Error("Pid signaled, pid: %v signal: %v", pid, s)
				delete(children, pid)
				continue
			}
			switch uint32(s) >> 8 {

			case uint32(unix.SIGTRAP) | (unix.PTRACE_EVENT_SECCOMP << 8):
				var regs syscall.PtraceRegs
				err = syscall.PtraceGetRegs(pid, &regs)

				if err != nil {
					log.Error("Error (ptrace): %v", err)
				}

				systemcall, err := syscallByNum(getSyscallNumber(regs))
				if err != nil {
					log.Error("Error: %v", err)
					continue
				}

				/* Render the system call invocation */

				r := getSyscallRegisterArgs(regs)
				call := ""

				if train == true {
					trainingset[getSyscallNumber(regs)] = true
					if systemcall.captureArgs != nil {
						for c, i := range systemcall.captureArgs {
							if i == 1 {
								if trainingargs[getSyscallNumber(regs)] == nil {
									trainingargs[getSyscallNumber(regs)] = make(map[int][]uint)
								}
								if contains(trainingargs[getSyscallNumber(regs)][c], uint(r[c])) == false {
									trainingargs[getSyscallNumber(regs)][c] = append(trainingargs[getSyscallNumber(regs)][c], uint(r[c]))
								}
							}
						}
					}
				}

				if f, ok := renderFunctions[getSyscallNumber(regs)]; ok {
					call, err = f(pid, r)
					if err != nil {
						log.Info("%v", err)
						continue
					}
					if *debug == true {
						call += "\n  " + renderSyscallBasic(pid, systemcall, regs)
					}
				} else {
					call = renderSyscallBasic(pid, systemcall, regs)
				}

				log.Info("seccomp hit on sandbox pid %v (%v) syscall %v (%v):\n  %s", pid, getProcessCmdLine(pid), systemcall.name, systemcall.num, call)
				continue

			case uint32(unix.SIGTRAP) | (unix.PTRACE_EVENT_EXIT << 8):
				if *debug == true {
					log.Error("Ptrace exit event detected pid %v (%s)", pid, getProcessCmdLine(pid))
				}
				delete(children, pid)
				continue

			case uint32(unix.SIGTRAP) | (unix.PTRACE_EVENT_CLONE << 8):
				newpid, err := syscall.PtraceGetEventMsg(pid)
				if err != nil {
					log.Error("PTrace event message retrieval failed: %v", err)
				}
				children[int(newpid)] = true
				if *debug == true {
					log.Error("Ptrace clone event detected pid %v (%s)", pid, getProcessCmdLine(pid))
				}
				continue
			case uint32(unix.SIGTRAP) | (unix.PTRACE_EVENT_FORK << 8):
				if *debug == true {
					log.Error("PTrace fork event detected pid %v (%s)", pid, getProcessCmdLine(pid))
				}
				newpid, err := syscall.PtraceGetEventMsg(pid)
				if err != nil {
					log.Error("PTrace event message retrieval failed: %v", err)
				}
				children[int(newpid)] = true
				continue
			case uint32(unix.SIGTRAP) | (unix.PTRACE_EVENT_VFORK << 8):
				if *debug == true {
					log.Error("Ptrace vfork event detected pid %v (%s)", pid, getProcessCmdLine(pid))
				}
				newpid, err := syscall.PtraceGetEventMsg(pid)
				if err != nil {
					log.Error("PTrace event message retrieval failed: %v", err)
				}
				children[int(newpid)] = true
				continue
			case uint32(unix.SIGTRAP) | (unix.PTRACE_EVENT_VFORK_DONE << 8):
				if *debug == true {
					log.Error("Ptrace vfork done event detected pid %v (%s)", pid, getProcessCmdLine(pid))
				}
				newpid, err := syscall.PtraceGetEventMsg(pid)
				if err != nil {
					log.Error("PTrace event message retrieval failed: %v", err)
				}
				children[int(newpid)] = true

				continue
			case uint32(unix.SIGTRAP) | (unix.PTRACE_EVENT_EXEC << 8):
				if *debug == true {
					log.Error("Ptrace exec event detected pid %v (%s)", pid, getProcessCmdLine(pid))
				}
				continue
			case uint32(unix.SIGTRAP) | (unix.PTRACE_EVENT_STOP << 8):
				if *debug == true {
					log.Error("Ptrace stop event detected pid %v (%s)", pid, getProcessCmdLine(pid))
				}
				continue
			case uint32(unix.SIGTRAP):
				if *debug == true {
					log.Error("SIGTRAP detected in pid %v (%s)", pid, getProcessCmdLine(pid))
				}
				continue
			case uint32(unix.SIGCHLD):
				if *debug == true {
					log.Error("SIGCHLD detected pid %v (%s)", pid, getProcessCmdLine(pid))
				}
				continue
			case uint32(unix.SIGSTOP):
				if *debug == true {
					log.Error("SIGSTOP detected pid %v (%s)", pid, getProcessCmdLine(pid))
				}
				continue
			case uint32(unix.SIGSEGV):
				if *debug == true {
					log.Error("SIGSEGV detected pid %v (%s)", pid, getProcessCmdLine(pid))
				}
				err = syscall.Kill(pid, 9)
				if err != nil {
					log.Error("kill: %v", err)
					os.Exit(1)
				}
				delete(children, pid)
				continue
			default:
				y := s.StopSignal()
				if *debug == true {
					log.Error("Child stopped for unknown reasons pid %v status %v signal %i (%s)", pid, s, y, getProcessCmdLine(pid))
				}
				continue
			}
		}

		if train == true {
			var u *user.User
			var e error
			u, e = user.Current()
			var resolvedpath = ""

			if e != nil {
				log.Error("user.Current(): %v", e)
			}

			if *trainingoutput != "" {
				resolvedpath = *trainingoutput
			} else {
				if *noprofile == false {
					resolvedpath, e = fs.ResolvePathNoGlob(p.Seccomp.Train_Output, u)
					if e != nil {
						log.Error("resolveVars(): %v", e)
					}
				} else {
					s := fmt.Sprintf("${HOME}/%s-%d.seccomp", fname(os.Args[2]), cpid)
					resolvedpath, e = fs.ResolvePathNoGlob(s, u)
				}
			}
			policyout := "execve:1\n"
			for call := range trainingset {
				done := false
				for c := range trainingargs {
					if c == call {
						for a, v := range trainingargs[c] {
							sc, _ := syscallByNum(call)
							policyout += fmt.Sprintf("%s:%s\n", sc.name, genArgs(uint(a), (v)))
							done = true
						}
					}
				}
				if done == false {
					sc, _ := syscallByNum(call)
					policyout += fmt.Sprintf("%s:1\n", sc.name)
				}
			}
			if *appendpolicy == true {
				log.Error("Not yet implemented.")
			}
			f, err := os.OpenFile(resolvedpath, os.O_CREATE|os.O_RDWR, 0600)
			if err == nil {
				_, err := f.WriteString(policyout)
				if err != nil {
					log.Error("Error writing policy file: %v", err)
				}
				err = f.Close()
				if err != nil {
					log.Error("Error closing policy file: %v", err)
				}
			} else {
				log.Error("Error opening policy file \"%s\": %v", resolvedpath, err)
			}
		}
	}
}

func genArgs(a uint, vals []uint) string {
	s := ""
	for idx, x := range vals {
		s += fmt.Sprintf(" arg%d == %d ", a, x)
		if idx < len(vals)-1 {
			s += "||"
		}
	}
	return s
}

func contains(slice []uint, val uint) bool {
	for _, x := range slice {
		if val == x {
			return true
		}
	}
	return false
}
func fname(p string) string {
	_, fname := path.Split(p)
	return fname
}
