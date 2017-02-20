package openvpn

import (
	"path"
	"bufio"
	"fmt"
	"os"
	"net"
	"os/exec"
	"regexp"
	"syscall"

	"github.com/subgraph/oz"
)

func StartOpenVPN(c *oz.Config, conf string, ip *net.IP, table, dev, auth, runtoken string) (cmd *exec.Cmd, err error) {


	confFile := path.Join(c.OpenVPNConfDir, conf)
	cmdArgs, err := parseOpenVPNConf(c, confFile, ip, table, dev, auth, runtoken)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error %v\n", err)
		return nil, err
	}

	runcmd := exec.Command("/usr/sbin/openvpn", cmdArgs...)
	runcmd.Stdin = os.Stdin
	runcmd.Stderr = os.Stderr

	/* 
	cmdReader, err := runcmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error runcmd.StdoutPipe(): %v\n", err)
	}
	scanner := bufio.NewScanner(cmdReader)

	go func() {
		for scanner.Scan() {
			fmt.Printf("Output: %s\n", scanner.Text())
		}
	}()
	*/

	runcmd.SysProcAttr = &syscall.SysProcAttr{}
	runcmd.SysProcAttr.Credential = &syscall.Credential{
		Gid: c.OpenVPNGID,
	}
	err = runcmd.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Error (exec): %v %s\n", err, cmdArgs[0])
		os.Exit(1)
	}
	return runcmd, nil

}

func parseOpenVPNConf(c *oz.Config, filename string, ip *net.IP, table, dev, auth, runtoken string) (cmdargs []string, err error) {

	var cmd []string
	var certpath, capath, keypath, tlsauthpath string
	pidfilepath := path.Join(c.OpenVPNRunPath, runtoken + ".pid")

	file, err := os.Open(filename)
	if err != nil {
		return []string{}, err
	}

	defer file.Close()

	r := regexp.MustCompile("[^\\s]+")
	reader := bufio.NewReader(file)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		x := r.FindAllString(scanner.Text(), -1)
		if len(x) == 0 {
			continue
		}

		switch x[0] {

		/* TODO: Need to review all OpenVPN client params and filter here */

		case "auth-user-pass":
			cmd = append(cmd, []string{"--auth-user-pass", path.Join(c.OpenVPNConfDir, auth)}...)
			continue
		case "route-up":
			continue
		case "route-pre-down":
			continue
		case "down":
			continue
		case "script-security":
			continue
		case "ifconfig":
			continue
		case "ca":
			if len(x) == 2 {
				cmd = append(cmd, []string{"--" + x[0], path.Join(c.OpenVPNConfDir, x[1])}...)
			}
			continue
		case "writepid":
			continue
		case "crl-verify":
			if len(x) == 2 {
				cmd = append(cmd, []string{"--" + x[0], path.Join(c.OpenVPNConfDir, x[1])}...)
			}
			continue
		case "<cert>":
			certpath = path.Join(c.OpenVPNRunPath, runtoken + "-cert.cert")
			f, err := os.Create(certpath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing cert to file: %v", err)
				return cmd, err
			}
			defer f.Close()
			for scanner.Scan() {
				if scanner.Text() == "</cert>" {
					f.Sync()
					break
				}
				_, err := f.WriteString(scanner.Text() + "\n")
				if err != nil {
					fmt.Fprintf(os.Stderr, "error writing cert contents to file: %v", err)
					return cmd, err
				}
			}
			cmd = append(cmd, []string{"--cert", certpath}...)
			continue
		case "<ca>":
			capath = path.Join(c.OpenVPNRunPath, runtoken + "-ca.cert")
			f, err := os.Create(capath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing cert to file: %v", err)
				return cmd, err
			}
			defer f.Close()
			for scanner.Scan() {
				if scanner.Text() == "</ca>" {
					f.Sync()
					break
				}
				_, err := f.WriteString(scanner.Text() + "\n")
				if err != nil {
					fmt.Fprintf(os.Stderr, "error writing cert contents to file: %v", err)
					return cmd, err
				}
			}
			cmd = append(cmd, []string{"--ca", capath}...)
			continue
		case "<key>":
			keypath = path.Join(c.OpenVPNRunPath, runtoken + "-key.key")
			f, err := os.Create(keypath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing key to file: %v", err)
				return cmd, err
			}
			defer f.Close()
			for scanner.Scan() {
				if scanner.Text() == "</key>" {
					f.Sync()
					break
				}
				_, err := f.WriteString(scanner.Text() + "\n")
				if err != nil {
					fmt.Fprintf(os.Stderr, "error writing key contents to file: %v", err)
					return cmd, err
				}
			}
			cmd = append(cmd, []string{"--key", keypath}...)
			continue
		case "<tls-auth>":
			tlsauthpath = path.Join(c.OpenVPNRunPath, runtoken + "-tls-auth.key")
			f, err := os.Create(tlsauthpath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing tls-auth to file: %v", err)
				return cmd, err
			}
			defer f.Close()
			for scanner.Scan() {
				if scanner.Text() == "</tls-auth>" {
					f.Sync()
					break
				}
				_, err := f.WriteString(scanner.Text() + "\n")
				if err != nil {
					fmt.Fprintf(os.Stderr, "error writing contents to file: %v", err)
					return cmd, err
				}
			}
			cmd = append(cmd, []string{"--tls-auth", tlsauthpath}...)
			continue
		default:
		}
		if len(x) == 1 {
			cmd = append(cmd, "--"+x[0])
		} else {
			cmd = append(cmd, "--"+x[0])
			for _, t := range x[1:] {
				cmd = append(cmd, t)
			}
		}
	}
	extra := []string{"--writepid", pidfilepath,"--daemon","--auth-retry", "nointeract", "--route-noexec", "--route-up", "/usr/bin/oz-ovpn-route-up", "--route-pre-down", "/usr/bin/oz-ovpn-route-down", "--script-security", "2", "--setenv", "bridge_addr", ip.String(), "--setenv", "routing_table", table, "--setenv", "bridge_dev", dev}
	cmd = append(cmd, extra...)

	for _, x := range cmd {
		fmt.Fprintf(os.Stderr, "%s", x)
		fmt.Fprintf(os.Stderr, " ")
	}
	return cmd, nil

}
