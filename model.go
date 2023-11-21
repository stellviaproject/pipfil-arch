package arch

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// This error is produced with panic when you call model with not enough or more than required length of arguments.
var ErrInputCountMismatch = errors.New("input count mismatch")

// This error is produced when in input or output pipes of model you have a pipe repeated
var ErrModelInOutRepeated = errors.New("model inout repeated")

// Join pipes into slice for easy filter and model creation
func WithPipes(pipes ...Pipe) []Pipe {
	return pipes
}

// Make pipe and length provided from other pipe association
func NewLen(pipe, len Pipe) Length {
	return &length{pipe, len}
}

// Pipe and length association for takeing pipe items and makeing a slice with length provided from other pipe
type Length interface {
	Pipe() Pipe //This is a pipe whose items will be set to slice
	Len() Pipe
}

type length struct {
	pipe   Pipe
	length Pipe
}

func (ln *length) Pipe() Pipe {
	return ln.pipe
}

func (ln *length) Len() Pipe {
	return ln.length
}

// Easy mode for creating a slice of Length association
func WithLens(lens ...Length) []Length {
	return lens
}

// Create a new filter from function and link it to corresponding input pipes and output pipes
func NewFilterWithPipes(name string, fn any, ins, outs []Pipe, lens []Length) Filter {
	fnc := FuncOf(fn)
	filter := NewFilter(name)
	filter.UseFunc(fnc)
	for i := range ins {
		fnc.In(ins[i])
		if err := filter.Input().SetNamed(ins[i]); err != nil {
			panic(err)
		}
		if err := ins[i].To(filter); err != nil {
			panic(err)
		}
	}
	for i := range lens {
		if err := filter.Input().SetLenFor(lens[i].Pipe(), lens[i].Len()); err != nil {
			panic(err)
		}
		if err := lens[i].Len().LenTo(lens[i].Pipe()); err != nil {
			panic(err)
		}
	}
	for i := range outs {
		fnc.Out(outs[i])
		if err := filter.Output().SetNamed(outs[i]); err != nil {
			panic(err)
		}
	}
	if err := filter.Compile(); err != nil {
		panic(err)
	}
	return filter
}

// Join filters for create easy model
func WithFilters(filters ...Filter) []Filter {
	return filters
}

// Join inputs to use in call to model
func WithInput(input ...any) []any {
	return input
}

// Represents a model with pipes-filters architecture
type Model interface {
	Call(input []any) []any         //Call model to evaluate in algorithm with pipes-filters architecture
	Run()                           //Run model
	Stop()                          //Stop model
	SetParallel(parallel int) error //Set parallel value to every filter
	Errs() []error                  //Get model error
	HasErrs() bool                  //Tell if model has errors
	PrintErrs()                     //Print errors
	Clear()                         //Clear model errors
}

type model struct {
	singal         Signal
	filters        []Filter
	inputs, outpus []Pipe
	inMap, outMap  map[string]int
	calls          []chan []any
	mtxIn, mtxOut  sync.Mutex
}

// Create a new model with pipes-filters architecture
func NewModel(filters []Filter, inputs, outpus []Pipe) Model {
	signal := &signal{}
	for i := range filters {
		filters[i].SetSignal(signal)
	}
	for i := range outpus {
		outpus[i].To(nil)
	}
	// deadlock detect
	// generate input map
	inIndex := make(map[string]int, len(inputs))
	inputMap := map[Pipe]int{}
	for i := range inputs {
		pipe := inputs[i]
		if _, ok := inputMap[pipe]; ok {
			panic(ErrModelInOutRepeated)
		}
		if _, ok := inIndex[pipe.Name()]; ok {
			panic(ErrModelInOutRepeated)
		}
		inputMap[pipe] = i
		inIndex[pipe.Name()] = i
	}
	// generate output map
	outIndex := make(map[string]int, len(outpus))
	outputMap := map[Pipe]int{}
	for i := range outpus {
		pipe := outpus[i]
		if _, ok := outputMap[pipe]; ok {
			panic(ErrModelInOutRepeated)
		}
		if _, ok := outIndex[pipe.Name()]; ok {
			panic(ErrModelInOutRepeated)
		}
		outputMap[pipe] = i
		outIndex[pipe.Name()] = i
	}
	for i := range filters {
		ftr := filters[i].(*filter)
		for input, length := range ftr.length {
			linkLen, linkOut := -1, -1
			var filterOutLen, filterOut *filter
			for j := range filters {
				ftr := filters[j].(*filter)
				if flnk, ok := ftr.outLink[length]; ok {
					if linkLen != -1 {
						panic(fmt.Errorf("pipe '%s' used as length is set as output to two filters", length.Name()))
					}
					linkLen = flnk
					filterOutLen = ftr
				}
				if flnk, ok := ftr.outLink[input]; ok {
					if linkOut != -1 {
						panic(fmt.Errorf("pipe '%s' used as input associated to pipe '%s' used as length is set as output to two filters", input.Name(), input.Name()))
					}
					linkOut = flnk
					filterOut = ftr
				}
			}
			fOutLenType := filterOutLen.fn.fnType.Out(linkLen)
			if fOutLenType.Kind() != reflect.Slice {
				panic(fmt.Errorf("pipe '%s' used as length is connected to filter '%s' output whose is not slice type", length.Name(), filterOutLen.name))
			}
			//deadlock condition
			fOutType := filterOut.fn.fnType.Out(linkOut)
			if input != length && fOutType != input.CheckType() {
				panic(fmt.Errorf("posible deadlock, pipe '%s' could not be used with a pipe '%s' as length because pipe '%s' is associated to a filter '%s' output for sending slice elements one by one", length.Name(), input.Name(), input.Name(), filterOut.name))
			}
		}
		ftr.Input().ForEach(func(in Pipe) bool {
			inCount, isModelInput := inputMap[in]
			if isModelInput {
				inputMap[in] = inCount + 1
			}
			isFilterOutput := false
			for j := range filters {
				if _, ok := filters[j].(*filter).outLink[in]; ok {
					if i == j {
						panic(fmt.Errorf("pipe '%s' is connected to filter '%s' as input and output at the same time", in.Name(), ftr.name))
					}
					isFilterOutput = true
					break
				}
			}
			if !isFilterOutput && !isModelInput {
				panic(fmt.Errorf("filter '%s' has input pipe '%s' not connected to model input or to other filter output", ftr.Name(), in.Name()))
			}
			return true
		})
		ftr.Output().ForEach(func(out Pipe) bool {
			outCount, isModelOutput := outputMap[out]
			if isModelOutput {
				outputMap[out] = outCount + 1
			}
			isFilterInput := false
			for j := range filters {
				if j != i && filters[j].Input().HasNamed(out.Name()) {
					isFilterInput = true
					break
				}
			}
			if !isFilterInput && !isModelOutput {
				panic(fmt.Errorf("filter '%s' has output pipe '%s' not connected to model output or to other filter input", ftr.Name(), out.Name()))
			}
			if isModelOutput {
				link := ftr.outLink[out]
				outType := ftr.outs[link]
				if outType != out.CheckType() {
					panic(fmt.Errorf("filter '%s' has output pipe '%s' as model output but it trys to send one by one", ftr.name, out.Name()))
				}
			}
			return true
		})
	}
	// check up input connection
	for pipe, count := range inputMap {
		if count == 0 {
			panic(fmt.Errorf("input pipe '%s' is not connected to a filter", pipe.Name()))
		}
	}
	// check up output connection
	for pipe, count := range outputMap {
		if count == 0 {
			panic(fmt.Errorf("output pipe '%s' is not connected to a filter", pipe.Name()))
		}
	}
	signal.stop = make(chan int, signal.count)
	return &model{
		singal:  signal,
		filters: filters,
		inputs:  inputs,
		outpus:  outpus,
		inMap:   inIndex,
		outMap:  outIndex,
		calls:   make([]chan []any, 0, 10),
	}
}

func (md *model) Errs() []error {
	errs := make([]error, 0, 10)
	for i := 0; i < len(md.filters); i++ {
		errs = append(errs, md.filters[i].Errs()...)
	}
	return errs
}

func (md *model) HasErrs() bool {
	for i := 0; i < len(md.filters); i++ {
		if md.filters[i].HasErrs() {
			return true
		}
	}
	return false
}

// Set parallel value to every filter
func (md *model) SetParallel(parallel int) error {
	for i := range md.filters {
		if err := md.filters[i].SetParallel(parallel); err != nil {
			return err
		}
	}
	return nil
}

// Call model with architecture of pipes and filters using an input
//
// Provided input will be redirected to every input of model in the same order.
//
// The order of output will be the same of provided order of output pipes in the model builder NewModel(...)
func (md *model) Call(input []any) []any {
	if len(input) != len(md.inputs) {
		panic(ErrInputCountMismatch)
	}

	md.mtxIn.Lock()
	//Critical section
	ch := make(chan []any, 1)
	md.calls = append(md.calls, ch)

	for i := 0; i < len(input); i++ {
		md.inputs[i].Set(input[i])
	}

	md.mtxIn.Unlock()
	wt := make(chan int, 1)
	go func() {
		//Critical section for outputs
		//Every gorutine is trying to get output at the same time
		md.mtxOut.Lock()
		output := make([]any, len(md.outpus))
		for i := 0; i < len(output); i++ {
			output[i] = md.outpus[i].Get(nil)
		}

		md.mtxIn.Lock()
		md.calls[0] <- output
		md.calls = md.calls[1:]
		md.mtxIn.Unlock()

		md.mtxOut.Unlock()
		wt <- 0
	}()
	output := <-ch
	<-wt
	return output
}

func (md *model) GetIn(in any) []any {
	inValue := reflect.ValueOf(in)
	inType := inValue.Type()
	ins := make([]any, len(md.inputs))
	for i := 0; i < inType.NumField(); i++ {
		field := inType.Field(i)
		if pipeName, ok := field.Tag.Lookup("pipe"); ok {
			if index, ok := md.inMap[pipeName]; ok {
				ins[index] = inValue.Field(i)
			} else {
				panic(fmt.Errorf("pipe '%s' not found", pipeName))
			}
		} else if index, ok := md.inMap[field.Name]; ok {
			ins[index] = inValue.Field(i)
		}
	}
	for i := 0; i < len(ins); i++ {
		if ins[i] == nil {
			panic(fmt.Errorf("pipe '%s' required value not found", md.inputs[i].Name()))
		}
	}
	return ins
}

func (md *model) SetOut(out any, outputs []any) {
	outValue := reflect.ValueOf(out)
	outType := outValue.Type()
	if outType.Kind() != reflect.Ptr {
		panic(fmt.Errorf("ptr to struct is required"))
	}
	outType = outType.Elem()
	outValue = outValue.Elem()
	for i := 0; i < outType.NumField(); i++ {
		field := outType.Field(i)
		var value any
		if pipeName, ok := field.Tag.Lookup("pipe"); ok {
			if index, ok := md.outMap[pipeName]; ok {
				value = outputs[index]
			} else {
				panic(fmt.Errorf("pipe '%s' not found", pipeName))
			}
		} else if index, ok := md.outMap[field.Name]; ok {
			value = outputs[index]
		}
		if value != nil {
			outValue.Field(i).Set(reflect.ValueOf(value))
		}
	}
}

// Run model
func (md *model) Run() {
	for i := range md.filters {
		go md.filters[i].Run()
	}
}

func (md *model) Clear() {
	for i := range md.filters {
		md.filters[i].Clear()
	}
}

// Wait for model stop or error
func (md *model) Wait() {
	md.singal.Wait()
}

// Stop model
func (md *model) Stop() {
	md.singal.Stop()
}

func (md *model) PrintErrs() {
	for _, err := range md.Errs() {
		fmt.Println(err)
	}
}
