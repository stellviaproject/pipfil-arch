package arch

import (
	"errors"
	"reflect"
)

var (
	ErrNameExist      = errors.New("param has name allready")
	ErrInOutHasNoName = errors.New("inout has no name")
	ErrIsNotFuncType  = errors.New("fn is not a func type")
)

type Function interface {
	NameIn(index int, name string) Function
	NameOut(index int, name string) Function
	In(pipe Pipe) Function
	Out(pipe Pipe) Function
	Compile() error
}

type function struct {
	fnType    reflect.Type
	method    reflect.Value
	ins       []string
	outs      []string
	inc, outc int
}

func FuncOf(fn any) Function {
	method := reflect.ValueOf(fn)
	fnType := method.Type()
	if fnType.Kind() != reflect.Func {
		panic(ErrIsNotFuncType)
	}
	return &function{
		fnType: fnType,
		method: method,
		ins:    make([]string, fnType.NumIn()),
		outs:   make([]string, fnType.NumOut()),
	}
}

func (fn *function) In(pipe Pipe) Function {
	fn.NameIn(fn.inc, pipe.Name())
	fn.inc++
	return fn
}

func (fn *function) Out(pipe Pipe) Function {
	fn.NameOut(fn.outc, pipe.Name())
	fn.outc++
	return fn
}

func (fn *function) NameIn(index int, name string) Function {
	for i := range fn.ins {
		if fn.ins[i] == name {
			panic(ErrNameExist)
		}
	}
	fn.ins[index] = name
	return fn
}

func (fn *function) NameOut(index int, name string) Function {
	for i := range fn.outs {
		if fn.outs[i] == name {
			panic(ErrNameExist)
		}
	}
	fn.outs[index] = name
	return fn
}

// Test repeated parameters are haveing a name
func (fn *function) Compile() error {
	for i := range fn.ins {
		if fn.ins[i] == "" {
			return ErrInOutHasNoName
		}
	}
	for i := range fn.outs {
		if fn.outs[i] == "" {
			return ErrInOutHasNoName
		}
	}
	return nil
}
