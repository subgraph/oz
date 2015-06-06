package ipc

import (
	"reflect"
	"syscall"
)

func setPassCred(c interface{}) error {
	fd := reflectFD(c)
	return syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_PASSCRED, 1)
}

func reflectFD(c interface{}) int {
	sysfd := extractField(c, "fd", "sysfd")
	return int(sysfd.Int())
}

func extractField(ob interface{}, fieldNames ...string) reflect.Value {
	v := reflect.Indirect(reflect.ValueOf(ob))
	for _, fn := range fieldNames {
		field := v.FieldByName(fn)
		v = reflect.Indirect(field)
	}
	return v
}
