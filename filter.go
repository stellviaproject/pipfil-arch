package arch

import (
	"errors"
	"reflect"
	"time"
)

// It's produced when filter has an error in its definition
var ErrFilterNotCompiled = errors.New("filter not checked")

// It's produced a pipe is allready in use in the collection, to avoid it use names for pipes and function arguments
var ErrPipeAllReadyInUse = errors.New("pipe allready in use, review the parameters of method if one type is duplicated and has no name")

// Represents a filter for pipes-filter architecture
type Filter interface {
	Name() string            //Filter name
	Input() PipeCollection   //Input collection of pipes linked to filter
	Output() PipeCollection  //Output collection of pipes linked to filter
	UseFunc(fn Function)     //Function that filter runs for processing pipes incoming data
	Compile() error          //Compile filter and test if it has errors in its definition
	SetSignal(signal Signal) //Set signal to control filter gorutines
	Run()                    //Run filter, it's must be run in a gorutine
}

type filter struct {
	name     string
	inLink   map[Pipe]int //Redirect data between pipe and method
	outLink  map[Pipe]int //Redirect data between method output and pipe
	length   map[Pipe]Pipe
	input    *collection
	output   *collection
	fn       *function
	sg       *signal
	compiled bool
}

func NewFilter(name string) Filter {
	return &filter{
		name:    name,
		input:   newCollection(),
		output:  newCollection(),
		inLink:  make(map[Pipe]int),
		outLink: make(map[Pipe]int),
		length:  make(map[Pipe]Pipe),
	}
}

func (ftr *filter) Name() string {
	return ftr.name
}

func (ftr *filter) UseFunc(fn Function) {
	ftr.fn = fn.(*function)
}

func (ftr *filter) Input() PipeCollection {
	return ftr.input
}

func (ftr *filter) Output() PipeCollection {
	return ftr.output
}

func (ftr *filter) Compile() error {
	fn := ftr.fn
	if err := fn.Compile(); err != nil {
		return err
	}
	for i := 0; i < len(fn.ins); i++ {
		inType := fn.fnType.In(i)
		if fn.ins[i] == "" {
			pipe, err := ftr.input.Get(inType)
			if err != nil {
				return err
			}
			if pipe.CheckType().Kind() != reflect.Slice && inType.Kind() == reflect.Slice {
				length, err := ftr.input.GetLenFor(pipe)
				if err != nil {
					return err
				}
				ftr.length[pipe] = length
			}
			if _, ok := ftr.inLink[pipe]; ok {
				return ErrPipeAllReadyInUse
			}
			ftr.inLink[pipe] = i
		} else {
			pipe, err := ftr.input.GetNamed(fn.ins[i])
			if err != nil {
				return err
			}
			if pipe.CheckType().Kind() != reflect.Slice && inType.Kind() == reflect.Slice {
				length, err := ftr.input.GetLenFor(pipe)
				if err != nil {
					return err
				}
				ftr.length[pipe] = length
			}
			ftr.inLink[pipe] = i
		}
	}
	for i := 0; i < len(fn.outs); i++ {
		outType := fn.fnType.Out(i)
		if fn.outs[i] == "" {
			pipe, err := ftr.output.Get(outType)
			if err != nil {
				return err
			}
			if outType != pipe.CheckType() && (outType.Kind() != reflect.Slice || outType.Elem() != pipe.CheckType()) {
				return ErrPipeTypeMismatch
			}
			if _, ok := ftr.outLink[pipe]; ok {
				return ErrPipeAllReadyInUse
			}
			ftr.outLink[pipe] = i
		} else {
			pipe, err := ftr.output.GetNamed(fn.outs[i])
			if err != nil {
				return err
			}
			if outType != pipe.CheckType() && (outType.Kind() != reflect.Slice || outType.Elem() != pipe.CheckType()) {
				return ErrPipeTypeMismatch
			}
			ftr.outLink[pipe] = i
		}
	}
	ftr.compiled = true
	return nil
}

func (ftr *filter) call(input []reflect.Value) (output []reflect.Value, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()
	output = ftr.fn.method.Call(input)
	last := output[len(output)-1].Interface()
	if err, ok := last.(error); ok {
		return output, err
	}
	return
}

func (ftr *filter) Run() {
	if !ftr.compiled {
		panic(ErrFilterNotCompiled)
	}
	sg := ftr.sg
LOOP:
	for ftr.input.IsOpen() && ftr.output.IsOpen() {
		if sg.stop != nil {
			select {
			case <-sg.stop:
				break LOOP
			default:
			}
		}
		input := make([]reflect.Value, len(ftr.fn.ins))
		ftr.input.ForEach(func(pipe Pipe) bool {
			index := ftr.inLink[pipe]
			length := ftr.length[pipe]
			if length != nil {
				sliceLen := length.Len(ftr)
				slice := reflect.MakeSlice(reflect.SliceOf(pipe.CheckType()), sliceLen, 10)
				for i := 0; i < sliceLen; i++ {
					slice.Index(i).Set(reflect.ValueOf(pipe.Get(ftr)))
				}
				input[index] = slice
			} else {
				input[index] = reflect.ValueOf(pipe.Get(ftr))
			}
			return true
		})
		if sg.stop != nil {
			select {
			case <-sg.stop:
				break LOOP
			default:
			}
		}
		output, err := ftr.call(input)
		if err != nil {
			sg.err <- err
		} else {
			ftr.output.ForEach(func(pipe Pipe) bool {
				index := ftr.outLink[pipe]
				out := output[index]
				if pipe.CheckType().Kind() != reflect.Slice && out.Kind() == reflect.Slice {
					//wrap slice
					pipe.SetLen(out.Len())
					for i := 0; i < out.Len(); i++ {
						pipe.Set(out.Index(i).Interface())
					}
				} else {
					pipe.Set(output[index].Interface())
				}
				return true
			})
		}
	}
	ftr.output.Close()
	ftr.input.Close()
}

func (ftr *filter) SetSignal(sg Signal) {
	ftr.sg = sg.(*signal)
	ftr.sg.count++
}

type Signal interface {
	Stop()
	Err() error
	Wait()
}

func NewSignal() Signal {
	return &signal{}
}

type signal struct {
	count    int
	stop     chan int
	err      chan error
	errValue error
}

func (sg *signal) Stop() {
	for i := 0; i < sg.count; i++ {
		sg.stop <- 0
	}
}

func (sg *signal) Wait() {
	if sg.count == 0 {
		return
	}
	sg.count++
	sg.stop = make(chan int, sg.count)
	sg.err = make(chan error, 1)
	wait := make(chan int)
	go func() {
		for {
			select {
			case sg.errValue = <-sg.err:
				wait <- 0
			case <-sg.stop:
				wait <- 0
			default:
				time.Sleep(time.Millisecond)
			}
		}
	}()
	<-wait
}

func (sg *signal) Err() error {
	return sg.errValue
}
