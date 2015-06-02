package ipc
import (
	"reflect"
	"errors"
	"fmt"
)

type handlerMap map[string]reflect.Value

func (handlers handlerMap) dispatch(m *Message) error {
	h,ok := handlers[m.Type]
	if !ok {
		return errors.New("no handler found for message type:"+ m.Type)
	}
	return executeHandler(h, m)
}

func executeHandler(h reflect.Value, m *Message) error {
	var args [2]reflect.Value
	args[0] = reflect.ValueOf(m.Body)
	args[1]= reflect.ValueOf(m)

	rs := h.Call(args[:])
	if len(rs) != 1 {
		return errors.New("handler function did not return expected single result value")
	}
	if rs[0].IsNil() {
		return nil
	}
	return rs[0].Interface().(error)
}

func (handlers handlerMap) addHandler(h interface{}) error {
	msgType, err := typeCheckHandler(h)
	if err != nil {
		return err
	}
	if _,ok := handlers[msgType]; ok{
		return fmt.Errorf("duplicate handler registered for message type '%s'", msgType)
	}
	handlers[msgType] = reflect.ValueOf(h)
	return nil
}

var errType = reflect.TypeOf((*error)(nil)).Elem()
var messageType = reflect.TypeOf((*Message)(nil))

func typeCheckHandler(h interface{}) (string, error) {
	t := reflect.TypeOf(h)
	if t.Kind() != reflect.Func {
		return "", fmt.Errorf("handler %v is not a function", t)
	}
	if t.NumIn() != 2 {
		return "", fmt.Errorf("handler %v has incorrect number of input arguments, got %d", t, t.NumIn())
	}
	if t.NumOut() != 1 {
		return "", fmt.Errorf("handler %v has incorrect number of return values %d", t, t.NumOut())
	}
	if t.In(0).Kind() != reflect.Ptr {
		return "", errors.New("first argument of handler is not a pointer")
	}
	in0 := t.In(0).Elem()
	if in0.Kind() != reflect.Struct {
		return "", fmt.Errorf("first argument of handler is not a pointer to struct")
	}
	if in1 := t.In(1); !in1.AssignableTo(messageType) {
		return "", fmt.Errorf("second argument of handler must have type *Message")
	}
	if out := t.Out(0); !out.AssignableTo(errType) {
		return "", fmt.Errorf("return type of handler must be error")
	}

	if in0.NumField() == 0 {
		return "", fmt.Errorf("first argument structure has no fields")
	}
	if len(in0.Field(0).Tag) == 0 {
		return "", fmt.Errorf("first argument structure, first field has no tag")
	}
	return string(in0.Field(0).Tag), nil
}


