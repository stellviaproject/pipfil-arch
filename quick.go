package arch

import "errors"

// This error is produced with panic when you call model with not enough or more than required length of arguments.
var ErrInputCountMismatch = errors.New("input count mismatch")

// Join pipes into slice for easy filter and model creation
func WithPipes(pipes ...Pipe) []Pipe {
	return pipes
}

func NewLen(pipe, len Pipe) Length {
	return &length{pipe, len}
}

type Length interface {
	Pipe() Pipe
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
		if err := lens[i].Len().LenTo(filter); err != nil {
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
	Err() error             //Get model error
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
	signal.stop = make(chan int, signal.count)
	return &model{
		singal:  signal,
		filters: filters,
		inputs:  inputs,
		outpus:  outpus,
	}
}

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

func (md *model) Run() {
	for i := range md.filters {
		go md.filters[i].Run()
	}
}

func (md *model) Wait() {
	md.singal.Wait()
}

func (md *model) Err() error {
	return md.singal.Err()
}

func (md *model) Stop() {
	md.singal.Stop()
}
