package arch

import (
	"errors"
	"fmt"

	//"fmt"
	"reflect"
	"sync"
)

// It's produced when filter has an error in its definition
var ErrFilterNotCompiled = errors.New("filter not checked")

// It's produced a pipe is allready in use in the collection, to avoid it use names for pipes and function arguments
var ErrPipeAllReadyInUse = errors.New("pipe allready in use, review the parameters of method if one type is duplicated and has no name")

// It's make panic when parallel is lesser than or equal to zero
var ErrParallelZeroNeg = errors.New("parallel is lesser than or equal to zero")

// Represents a filter for pipes-filter architecture
type Filter interface {
	Name() string                   //Filter name
	Input() PipeCollection          //Input collection of pipes linked to filter
	Output() PipeCollection         //Output collection of pipes linked to filter
	UseFunc(fn Function)            //Function that filter runs for processing pipes incoming data
	Compile() error                 //Compile filter and test if it has errors in its definition
	SetSignal(signal Signal)        //Set signal to control filter gorutines
	SetParallel(parallel int) error //Control number of filter gorutines for processing multiple inputs at the same time
	Run()                           //Run filter, it's must be run in a gorutine
	Clear()                         //Clear errors
	Errs() []error                  //Return internal error list
	HasErrs() bool                  //Tell if there are errors
	PrintErrs()                     //Print errors
}

type filter struct {
	name     string
	inLink   map[Pipe]int //Redirect data between pipe and method
	outLink  map[Pipe]int //Redirect data between method output and pipe
	length   map[Pipe]Pipe
	input    *collection
	output   *collection
	fn       *function
	outs     []reflect.Type
	errs     []error
	parallel int
	sg       *signal
	lck      chan int
	q        *queue
	compiled bool
}

func NewFilter(name string) Filter {
	return &filter{
		name:     name,
		input:    newCollection(),
		output:   newCollection(),
		inLink:   make(map[Pipe]int),
		outLink:  make(map[Pipe]int),
		length:   make(map[Pipe]Pipe),
		errs:     make([]error, 0, 10),
		lck:      make(chan int, 1),
		parallel: 1,
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

func (ftr *filter) SetParallel(parallel int) error {
	if parallel <= 0 {
		return ErrParallelZeroNeg
	}
	ftr.parallel = parallel
	return nil
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
			if _, ok := ftr.inLink[pipe]; ok {
				return ErrPipeAllReadyInUse
			}
			ftr.inLink[pipe] = i
		} else {
			pipe, err := ftr.input.GetNamed(fn.ins[i])
			if err != nil {
				return err
			}

			if pipe.CheckType() != inType && (inType.Kind() != reflect.Slice || pipe.CheckType() != inType.Elem()) {
				return fmt.Errorf("filter '%s' has input pipe '%s' of type '%s' linked to type '%s'", ftr.name, pipe.Name(), pipe.CheckType(), inType)
			}
			if inType.Kind() == reflect.Slice && pipe.CheckType() == inType.Elem() {
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
			if _, ok := ftr.outLink[pipe]; ok {
				return ErrPipeAllReadyInUse
			}
			ftr.outLink[pipe] = i
		} else {
			pipe, err := ftr.output.GetNamed(fn.outs[i])
			if err != nil {
				return err
			}
			if outType != pipe.CheckType() && (outType.Kind() != reflect.Slice || pipe.CheckType() != outType.Elem()) {
				return fmt.Errorf("filter '%s' has input pipe '%s' of type '%s' linked to type '%s'", ftr.name, pipe.Name(), pipe.CheckType(), outType)
			}
			ftr.outLink[pipe] = i
		}
	}
	ftype := fn.fnType
	ftr.outs = make([]reflect.Type, ftype.NumOut())
	for i := 0; i < ftype.NumOut(); i++ {
		ftr.outs[i] = ftype.Out(i)
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
		return nil, err
	}
	return output, nil
}

func (ftr *filter) Clear() {
	ftr.errs = make([]error, 0, 10)
}

func (ftr *filter) Errs() []error {
	return ftr.errs
}

func (ftr *filter) Run() {
	if !ftr.compiled {
		panic(ErrFilterNotCompiled)
	}
	ftr.q = NewQueue(ftr.parallel)
	ftr.q.run(func(v any) {
		msg := v.(*msg)
		ftr.send(msg.output, msg.err, msg.unset)
	})
	ftr.errs = make([]error, 0, 10)
	sg := ftr.sg
	for ftr.input.IsOpen() && ftr.output.IsOpen() {
		if sg.tryStop() {
			break
		}
		input := make([]reflect.Value, len(ftr.fn.ins))
		unset := false
		wg := sync.WaitGroup{}
		ftr.input.ForEach(func(pipe Pipe) bool {
			wg.Add(1)
			go func() {
				defer wg.Done()
				index := ftr.inLink[pipe]
				length := ftr.length[pipe]
				if length != nil {
					//fmt.Println(ftr.name, " <- Len ", pipe.Name())
					sliceLen := length.Len(pipe)
					slice := reflect.MakeSlice(reflect.SliceOf(pipe.CheckType()), sliceLen, sliceLen)
					for i := 0; i < sliceLen; i++ {
						//fmt.Println(ftr.name, " [", i, "] <- ", pipe.Name())
						slice.Index(i).Set(reflect.ValueOf(pipe.Get(ftr)))
					}
					input[index] = slice
				} else {
					//fmt.Println(ftr.name, " <- ", pipe.Name())
					value := pipe.Get(ftr)
					if value != nil {
						input[index] = reflect.ValueOf(value)
					} else {
						unset = true
					}
				}
			}()
			return true
		})
		wg.Wait()
		if sg.tryStop() {
			break
		}
		if ftr.parallel > 1 {
			ch := ftr.q.push(input)
			go ftr.process(input, ch, unset)
		} else {
			ftr.send(ftr.process(input, nil, unset))
		}
	}
	ftr.output.Close()
	ftr.input.Close()
	ftr.q.exit()
}

type msg struct {
	output []reflect.Value
	err    error
	unset  bool
}

func (ftr *filter) process(input []reflect.Value, send chan any, unset bool) ([]reflect.Value, error, bool) {
	var output []reflect.Value
	var err error
	if !unset {
		output, err = ftr.call(input)
		if err != nil {
			ftr.lck <- 0
			ftr.errs = append(ftr.errs, err)
			<-ftr.lck
		}
	}
	if send != nil {
		send <- &msg{
			output: output,
			err:    err,
			unset:  unset,
		}
		ftr.q.set()
	}
	return output, err, unset
}

func (ftr *filter) send(output []reflect.Value, err error, unset bool) {
	wg := sync.WaitGroup{}
	ftr.output.ForEach(func(pipe Pipe) bool {
		wg.Add(1)
		go func() {
			defer wg.Done()
			index := ftr.outLink[pipe]
			otype := ftr.outs[index]
			if otype.Kind() == reflect.Slice && pipe.CheckType() == otype.Elem() {
				if err != nil || unset {
					pipe.SetLen(0)
				} else {
					out := output[index]
					pipe.SetLen(out.Len())
					for i := 0; i < out.Len(); i++ {
						pipe.Set(out.Index(i).Interface())
					}
				}
			} else {
				if err != nil || unset {
					pipe.Set(nil)
				} else {
					pipe.Set(output[index].Interface())
				}
			}
		}()
		return true
	})
	wg.Wait()
}

func (ftr *filter) SetSignal(sg Signal) {
	ftr.sg = sg.(*signal)
	ftr.sg.count++
}

func (ftr *filter) HasErrs() bool {
	return len(ftr.errs) > 0
}

func (ftr *filter) PrintErrs() {
	for i := range ftr.errs {
		fmt.Println(ftr.errs[i])
	}
}

// Used to control the execution of filters from goroutines.
type Signal interface {
	Stop() //Stops the execution of the filters
	Wait() //Wait for all the filters to finish their execution.
}
