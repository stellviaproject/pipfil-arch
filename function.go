package arch

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrNameExist      = errors.New("param has name allready")
	ErrInOutHasNoName = errors.New("inout has no name")
	ErrIsNotFuncType  = errors.New("fn is not a func type")
)

// Represents a function used by filter
type Function interface {
	NameIn(index int, name string) Function  //Set a name to a function call parameter
	NameOut(index int, name string) Function //Set a name to a function return parameter
	In(pipe Pipe) Function                   //Adds a sequentially associated pipe to the function call parameters.
	Out(pipe Pipe) Function                  //Adds a sequentially associated pipe to the function return parameters.
	Compile() error                          //Check if there are any errors in the definition
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
	inTypes := map[reflect.Type]int{}
	outTypes := map[reflect.Type]int{}
	for i := 0; i < fn.fnType.NumIn(); i++ {
		curr := fn.fnType.In(i)
		if fn.ins[i] == "" {
			inTypes[curr]++
			if inTypes[curr] > 1 {
				return fmt.Errorf("input parameter of type '%s' in position %d has no name and its type is allready in use.", curr.String(), i)
			}
		}
	}
	for i := 0; i < fn.fnType.NumOut(); i++ {
		curr := fn.fnType.In(i)
		if fn.outs[i] == "" {
			outTypes[curr]++
			if outTypes[curr] > 1 {
				return fmt.Errorf("output parameter of type '%s' in position %d has no name and its type is allready in use.", curr.String(), i)
			}
		}
	}
	return nil
}
