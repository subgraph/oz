package ipc

import (
	"errors"
	"reflect"
	"testing"
)

func TestTypeCheckHandler(t *testing.T) {
	type testStruct struct{}

	cases := []interface{}{
		"foo",
		func() {},
		func(a, b int) {},
		func(a *testStruct, b *Message) {},
		func(a, b *int) error { return nil },
		func(a *testStruct, b int) error { return nil },
		func(a *testStruct, b Message) error { return nil },
		func(a *testStruct, b *Message) int { return 0 },
		func(a *testStruct, b *Message, c int) error { return nil },
	}

	for i, h := range cases {
		if _, err := typeCheckHandler(h); err == nil {
			t.Errorf("typeCheckHandler should return an error for case %d", i)
		}
	}
}

func TestAddHandler(t *testing.T) {
	hmap := handlerMap(make(map[string]reflect.Value))
	type testStruct struct {
		t int "tst"
	}
	legit := func(ts *testStruct, m *Message) error { return nil }

	if err := hmap.addHandler("bar"); err == nil {
		t.Error("attempt to register string as handler function did not fail as expected")
	}

	if err := hmap.addHandler(legit); err != nil {
		t.Error("registration of good handler function failed:", err)
	}

	if err := hmap.addHandler(legit); err == nil {
		t.Error("registration of duplicate handler function did not fail")
	}
}

func TestDispatch(t *testing.T) {
	type testStruct struct {
		t int "tester"
	}
	type testStruct2 struct {
		t int "tester2"
	}
	count := 0
	h1 := func(ts *testStruct, m *Message) error {
		count += 1
		return nil
	}

	h2 := func(ts *testStruct2, m *Message) error {
		count += 1
		return errors.New("...")
	}

	hmap := handlerMap(make(map[string]reflect.Value))
	if err := hmap.addHandler(h1); err != nil {
		t.Errorf("unexpected failure to register handler: %v", err)
	}
	if err := hmap.addHandler(h2); err != nil {
		t.Errorf("unexpected failure to register handler: %v", err)
	}
	m := new(Message)
	m.Type = "tester"
	m.Body = new(testStruct)
	if err := hmap.dispatch(m); err != nil {
		t.Error("unexpected error calling dispatch():", err)
	}
	m.Type = "tester2"
	m.Body = new(testStruct2)
	if err := hmap.dispatch(m); err == nil {
		t.Errorf("dispatch() did not return error as expected")

	}
	if count != 2 {
		t.Errorf("count was not incremented to 2 as expected. count = %d", count)
	}
}
