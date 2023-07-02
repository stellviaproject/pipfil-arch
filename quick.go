package arch

import (
	"errors"
	"fmt"
	"reflect"
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
	Call(input []any) []any //Call model to evaluate in algorithm with pipes-filters architecture
	Run()                   //Run model
	Stop()                  //Stop model

	//TODO: This function is disable for debuging
	//SetParallel(parallel int) error //Set parallel value to every filter

	Errs() []error //Get model error
	HasErrs() bool //Tell if model has errors
	PrintErrs()    //Print errors
	Clear()        //Clear model errors
}

type model struct {
	singal         Signal
	filters        []Filter
	inputs, outpus []Pipe
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
	inputMap := map[Pipe]int{}
	for i := range inputs {
		pipe := inputs[i]
		if _, ok := inputMap[pipe]; ok {
			panic(ErrModelInOutRepeated)
		}
		inputMap[pipe] = 0
	}
	// generate output map
	outputMap := map[Pipe]int{}
	for i := range outpus {
		pipe := outpus[i]
		if _, ok := outputMap[pipe]; ok {
			panic(ErrModelInOutRepeated)
		}
		outputMap[pipe] = 0
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

//TODO: This function is disable for debuging
// Set parallel value to every filter
// func (md *model) SetParallel(parallel int) error {
// 	for i := range md.filters {
// 		if err := md.filters[i].SetParallel(parallel); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// Call model with architecture of pipes and filters using an input
//
// Provided input will be redirected to every input of model in the same order.
//
// The order of output will be the same of provided order of output pipes in the model builder NewModel(...)
func (md *model) Call(input []any) []any {
	if len(input) != len(md.inputs) {
		panic(ErrInputCountMismatch)
	}
	for i := 0; i < len(input); i++ {
		md.inputs[i].Set(input[i])
	}
	output := make([]any, len(md.outpus))
	for i := 0; i < len(output); i++ {
		output[i] = md.outpus[i].Get(nil)
	}
	return output
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
