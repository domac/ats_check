package app

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"syscall"
)

const (
	EXIT = "exit"
	WAIT = "wait"
)

var (
	Events = make(map[string][]func(interface{}), 2)
)

func On(name string, fs ...func(interface{})) error {
	evs, ok := Events[name]
	if !ok {
		evs = make([]func(interface{}), 0, len(fs))
	}

	for _, f := range fs {
		fp := reflect.ValueOf(f).Pointer()
		for i := 0; i < len(evs); i++ {
			if reflect.ValueOf(evs[i]).Pointer() == fp {
				return fmt.Errorf("func[%v] already exists in event[%s]", fp, name)
			}
		}
		evs = append(evs, f)
	}
	Events[name] = evs
	return nil
}

func Emit(name string, arg interface{}) {
	evs, ok := Events[name]
	if !ok {
		return
	}

	for _, f := range evs {
		f(arg)
	}
}

func EmitAll(arg interface{}) {
	for _, fs := range Events {
		for _, f := range fs {
			f(arg)
		}
	}
	return
}

func Off(name string, f func(interface{})) error {
	evs, ok := Events[name]
	if !ok || len(evs) == 0 {
		return fmt.Errorf("envet[%s] doesn't have any funcs", name)
	}

	fp := reflect.ValueOf(f).Pointer()
	for i := 0; i < len(evs); i++ {
		if reflect.ValueOf(evs[i]).Pointer() == fp {
			evs = append(evs[:i], evs[i+1:]...)
			Events[name] = evs
			return nil
		}
	}

	return fmt.Errorf("%v func dones't exist in event[%s]", fp, name)
}

func OffAll(name string) error {
	Events[name] = nil
	return nil
}

func Wait(sig ...os.Signal) os.Signal {
	c := make(chan os.Signal, 1)
	if len(sig) == 0 {
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	} else {
		signal.Notify(c, sig...)
	}
	return <-c
}
