package seccomp

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func render_inetaddr(buf []byte) string {

	var fam uint16
	var port uint16

	fambuf := bytes.NewBuffer(buf[:2])
	binary.Read(fambuf, binary.LittleEndian, &fam)
	portbuf := bytes.NewBuffer(buf[2:4])
	binary.Read(portbuf, binary.BigEndian, &port)

	ipstr := fmt.Sprintf("%d.%d.%d.%d", byte(buf[4]), byte(buf[5]), byte(buf[6]), byte(buf[7]))

	structrep := fmt.Sprintf("{sin_family=%s, sin_port=%d, sin_addr=%s}", domainflags[uint(fam)], port, ipstr)
	return structrep
}

func render_unixaddr(buf []byte) string {

	var fam uint16

	fambuf := bytes.NewBuffer(buf[:2])
	binary.Read(fambuf, binary.LittleEndian, &fam)
	i := 0
	abstract := false

	if buf[2] == '\x00' {
		abstract = true
	}

	for i = 3; i < len(buf); i++ {
		if buf[i] == '\x00' {
			break
		}
	}
	name := buf[2:i]

	if abstract == true {
		name[0] = '@'
	}
	strname := string(name)

	return fmt.Sprintf("{sin.family=%s,sun_path=\"%s\"}", domainflags[uint(fam)], strname)
}
