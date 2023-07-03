# Documentation
## Introduction
This library was created for easy implementation of projects or algorithms with a pipe and filter architecture in go. There are still things to test, but it can be said that it is usable for the features that are enabled, of which it can be said that they have already been tested.
The features that can be used are:
- Creation of pipes with any type of data in go.
- Creation of filters with any type of function in go (Not yet tested for functions that do not return anything, this is under the assumption that each filter receives an input and returns an output).
- Connection between the filters giving each filter an input pipe and an output pipe that correspond to the data types of the functions in the same order or according to the names given to the parameters of the function (which must be specified in the creation of the object representing the function because go does not preserve parameter names when compiled).
- Sending of data through a pipe in parallel from the output of each filter or from the input of the architecture that is built.
- I receive data at the entrance of each filter from a pipe.
- Each filter waits for the function inputs from each pipe to complete.
- Sending slices through a pipe that is the data type of the elements of that slice, the elements will be sent one by one and the pipe will specify the number of elements to be sent.
- Construction of a slice with the input elements of a pipe by specifying another pipe that sends the number of elements.
- Construction of a model that represents an architecture of pipes and filters.
- Checking the conditions that could produce a deadlock in the model when executed.
- Sending the data through the model as if it were calling a function (the data can be sent in parallel).

These are the functionalities that have not been implemented due to difficulties in dedicating time to the library and due to the difficulty in debugging it:
- Processing of multiple inputs in parallel and sending the results in the same order as the corresponding inputs in the output.
#### Note:
About this, it has to be said that the code is present in the repository of the library. But the functions that allow you to specify the number of gorutins in parallel that each filter can execute have not been enabled. These functions are disabled and have a TODO to mark them. It would be appreciated if someone enables them and tries to improve or make the code work. Even so, maybe at some point in the future an attempt will be made to fix that part.

## Library Items
| Name |   Type    | Description|
|------|-----------|------------|
| Pipe | interface | Represents a pipeline through which data can be sent to the input of a filter, from one filter to another filter, or from a filter to the output of the architecture. |
|Filter| interface | Represents a filter formed from a function to process data received from a pipe. The input parameters of the function must be joined with pipes that have the same data types or if an input parameter is a slice it can be joined with a pipe that is not a slice but of the same data type of the elements of the slice, under the condition of specifying a pipe that provides the number of elements using the LenTo(pipe Pipe) error function. It must be taken into account that this pipe cannot be connected to the output of a filter that sends a slice, otherwise a deadlock will be obtained when executing; this is in custom models without using the NewModel(...) function which allows detection of a possible deadlock. The output of the filter can be specified using a pipe that has the same data type as the return of the function or in case a slice is returned, a pipe of the data type of the elements of that slice can be specified to send each element through the pipe. It's necesary to say that you must not use a Pipe for error type in the last return argument of a function, because the filter take that error and send it to Signal for stoping every filter |
| PipeCollection | interface | It is used to specify the input and output pipes in a filter. It has two ways of specifying it, one is using the data type and the other is the name of the pipe. First, when using the data type, you specify the data type of the pipe as the same as the function (either in the call or return parameters) and you are not allowed to use slices to connect them to pipes that are not slices (this condition is strict). The second form uses the names specified in a pipe to indicate the inputs or outputs of a filter. Note that specifying it in this method only indicates the pipes that the filter will use but does not literally join the input pipes to the filter (for which you must use the To(filter Filter) error method of the Pipe interface). |
| Function | interface | Represents the function that processes the filter data. It is used to name each of the call and return parameters sequentially. These names must match the names of the input and output pipes specified in the filter. |
| Signal | interface | This interface is used to control the execution of the gorutines inside the filters and stoping filters. It can also be used to wait for the execution of all the filters (until the Stop method of Signal is called somewhere in the code). |
| Length | interface | It represents the link between a pipeline that provides the elements and another pipeline that provides the number of elements in such a way that it is possible to build a slice with those elements for an input of the filter function. |
| FuncOf(fn any) Function | function | Creates the Function interface that represents a function. **Note:** There is no check at this time that the function has any returns, but it must in order to be piped (this is specified to avoid errors because this part has not been tested) |
| WithPipes(pipes ...Pipe) []Pipe | function | It's an easy way to join multiple pipes into a slice to pass as inputs or outputs to the function that creates the filter. |
| NewLen(pipe Pipe, len Pipe) Length | function | It is a function that receives as a parameter a pipe for the data and another pipe that will specify how many elements will be used in the input of a filter to build a slice from the elements of the pipe. |
| WithLens(lens ...Length) []Length | function | It's an easy way to create a slice of the Length interface to use in the function that creates the filters. |
| NewFilter(name string) Filter | function | It is a function that creates a filter without any pipes attached to its input or output, and without any functions that process the data. |
| NewFilterWithPipes(name string, fn any, ins, outs []Pipe, lens []Length) Filter | function | It is a function that creates a filter with a name, with the function that processes the data, with the input and output pipes, as well as the junctions between the pipes that provide the elements and those that provide the quantities to build a slice. The order of the elements in the input and output pipes must be the same order as the call and return elements of the function, without specifying a pipe for the error in the last parameter. |
| NewSignal() Signal | function | Create the Signal interface to control the filter goroutines. |
| func WithFilters(filters ...Filter) []Filter | function | This is a function to easily join a set of filters into a slice.|
| NewModel(filters []Filter, inputs, outpus []Pipe) Model | function | This is a function that creates a pipe and filter architecture model that can be called with the Call method as if it were a function. This function checks if a deadlock will occur when running the model, so it is recommended to use it to create the proposed architectures. |
| WithInput(input ...any) []any | function | This is a function to join a set of elements of any type into a slice[]any that can be used to run the model with the Call function |

### Interface Methods
---
#### Interface Pipe
| Methods | Description |
|---------|-------------|
| To(filter Filter) error | Specifies that a filter will receive items from the pipeline. Returns an error if it has already been specified. |
| LenTo(pipe Pipe) error | Specifies that a pipe will use another pipe to receive the number of elements with which a filter can build a slice. |
| Set(data any) | Sends an element through the pipeline. |
| Get(filter Filter) any | Receives an element through the pipeline at the specified filter. If the pipe was not specified to send data to the filter with the To(filter Filter) error method, a panic will fail. |
| SetLen(len int) | Sends the number of items down the pipeline so that filters can build slices from items in the same pipeline or from another pipeline that generates the same number of items. |
| Len(pipe Pipe) int | Gets the number of pipelined items associated with a pipeline. |
| CheckType() reflect.Type | Gets the data type of the items being sent through the pipeline. |
| IsOpen() bool | Determines whether the pipe channels are open or closed. |
| Close() | Close all channels of the pipeline. This will also terminate any filters associated with the pipe either as input or output.|
---
#### Interface Filter
| Method | Description |
|-|-|
| Name() string | The name of the filter. |
| Input() PipeCollection | Collection of the filter inlet pipes. |
| Output() PipeCollection | Collection of the filter outlet pipes. |
| UseFunc(fn Function) | Function that processes the data from the filter inlet pipes. |
| Compile() error | Analyzes the construction of the filter to find possible errors in its definition. It performs the binding of the input pipes with the call parameters of the function that the filter executes, as well as the binding of the return parameters with the output pipes. Determines if an input parameter that is a slice is to be built from an input pipe and another that specifies its number of elements, or if an output pipe is to be used to specify the number of slices. |
| SetSignal(signal Signal) | Sets the interface that controls the execution of the filter in parallel, determining if it stops when calling Stop or if an error occurs. |
| SetParallel(parallel int) error | (<span style="color:red">Disabled with comments</span>) Control number of filter gorutines for processing multiple inputs at the same time |
| Run() | Run the filter, it's must be run in a gorutine |
| Errs() []error | Return filter error list. |
| HasErrs() bool | Tell if the filter has errors. |
| PrintErrs() | Print filter errors. |
---
#### Interface Collection

| Methods | Description |
|-|-|
| Set(pipe Pipe) error  | Sets a pipe using its data type. The filter will bind the pipe to a function call or return parameter according to the data type at the parameter position. It should be noted that if the function has more than one element with the same data type on call or return an error will occur with panic. |
| SetNamed(pipe Pipe) error | Establishes a pipe using its name. This is the most recommended way since it is possible to use it with functions that have several call and return parameters with the same data type. |
| Get(pipeType reflect.Type) (Pipe, error) | Get a pipe by type. |
| GetNamed(name string) (Pipe, error) | Get a pipe by name. |
| ForEach(action func(Pipe) bool) bool | Run action for every pipe. |
| GetLenFor(pipe Pipe) (Pipe, error) | Get pipe that provides length for pipe. |
| SetLenFor(pipe, length Pipe) error | Set a pipe that provides length for pipe. |
| Has(pipeType reflect.Type) bool | It Tells if collection has a pipe for type.
| HasNamed(name string) bool | It tells if collection has a pipe with name. |
| IsOpen() bool | It tells if every pipe is open. |
| Close() | Close all pipes. |
---
#### Interface Function

| Methods | Description |
|-|-|
| NameIn(index int, name string) Function | Set a name to a function call parameter. |
| NameOut(index int, name string) Function | Set a name to a function return parameter. |
| In(pipe Pipe) Function | Adds a sequentially associated pipe to the function call parameters. |
| Out(pipe Pipe) Function | Adds a sequentially associated pipe to the function return parameters. |
| Compile() error | Check if there are any errors in the definition. |
---
#### Interface Signal

| Methods | Description |
|-|-|
| Stop() | Stops the execution of the filters. |
| Wait() | Wait for all the filters to finish their execution. |
---
#### Interface Model

| Methods | Description |
|-|-|
| Call(input []any) []any | Calls the model by passing the input values to the corresponding pipes and gets the results from the output pipes in the order specified when they were created. |
| Run() | Run the model by running each of its filters. |
| Stop() | Stops the execution of the model. |
| SetParallel(parallel int) error | (Disabled with comments) Sets the number of gorutines to use in parallel to process the inputs. |
| Errs() []error | Gets the model execution errors if any. |
| HasErrs() bool | Tell if model has errors. |
| PrintErrs() | Print model errors. |
| Clear() | Clear model errors. |
## Examples
#### 1- Create pipes, filters, signal and prepare a custom architecture.

![Example of pipes and filter architecture](https://github.com/stellviaproject/pipfil-arch/tree/master/example/custom/custom.png)

```golang
package main

import (
	"fmt"
	"math"

	arch "github.com/stellviaproject/pipfil-arch"
)

func main() {
	// 1st - create pipes
	input := arch.NewPipe("input", int(0), 1)
	duplicated := arch.NewPipe("duplicated", int(0), 1)
	triplicated := arch.NewPipe("triplicated", int(0), 1)
	squared := arch.NewPipe("squared", float64(0), 1)
	cubed := arch.NewPipe("cubed", float64(0), 1)
	loged := arch.NewPipe("loged", float64(0), 1)
	tripXsquared := arch.NewPipe("3xsqrt", float64(0), 1)
	output := arch.NewPipe("output", float64(0), 1)

	// 2nd - Create filters
	duplicate := arch.NewFilter("duplicate")
	triplicate := arch.NewFilter("triplicate")
	square := arch.NewFilter("square")
	tripXsquare := arch.NewFilter("3xsquare")
	logxcub := arch.NewFilter("multiple")
	substract := arch.NewFilter("substract")

	// 3rd - Set filter functions
	duplicate.UseFunc(arch.FuncOf(func(input int) int {
		return 2 * input
	}).In(input).Out(duplicated), //Link function input and outputs to a pipe
	)

	triplicate.UseFunc(arch.FuncOf(func(input int) int {
		return 3 * input
	}).In(duplicated).Out(triplicated)) //Link function input and outputs to a pipe

	square.UseFunc(arch.FuncOf(func(input int) float64 {
		return math.Sqrt(float64(input))
	}).In(duplicated).Out(squared)) //Link function input and outputs to a pipe

	tripXsquare.UseFunc(arch.FuncOf(func(triplicated int, squared float64) float64 {
		return float64(triplicated) * squared
	}).In(triplicated).In(squared).Out(tripXsquared)) //Link function inputs and outputs to a pipe

	logxcub.UseFunc(arch.FuncOf(func(triplicated int, squared float64) (float64, float64) {
		return math.Pow(float64(triplicated)*squared, 3), math.Log(float64(triplicated) * squared)
	}).In(triplicated).In(squared).Out(cubed).Out(loged)) //Link function inputs and outputs to a pipe

	substract.UseFunc(arch.FuncOf(func(cubed, loged, tripxsquare float64) float64 {
		return cubed - loged - tripxsquare
	}).In(cubed).In(loged).In(tripXsquared).Out(output)) //Link function inputs and outputs to a pipe

	//4th - Set pipes inputs and outputs in filter collection
	duplicate.Input().SetNamed(input)       //Set input pipe
	duplicate.Output().SetNamed(duplicated) //Set output pipe

	triplicate.Input().SetNamed(duplicated)   //Set input
	triplicate.Output().SetNamed(triplicated) //Set output pipe

	square.Input().SetNamed(duplicated) //Set input pipe
	square.Output().SetNamed(squared)   //Set output pipe

	tripXsquare.Input().SetNamed(triplicated)   //Set input pipe
	tripXsquare.Input().SetNamed(squared)       //Set input pipe
	tripXsquare.Output().SetNamed(tripXsquared) //Set output pipe

	logxcub.Input().SetNamed(triplicated) //Set input pipe
	logxcub.Input().SetNamed(squared)     //Set input pipe
	logxcub.Output().SetNamed(cubed)      //Set output pipe
	logxcub.Output().SetNamed(loged)      //Set output pipe

	substract.Input().SetNamed(cubed)        //Set input pipe
	substract.Input().SetNamed(loged)        //Set input pipe
	substract.Input().SetNamed(tripXsquared) //Set input pipe
	substract.Output().SetNamed(output)      //Set output pipe

	//5th - Redirect pipe data to filters
	input.To(duplicate)         //input pipe to duplicate filter
	duplicated.To(triplicate)   //duplicated pipe to triplicate filter
	duplicated.To(square)       //duplicated pipe to square filter
	triplicated.To(tripXsquare) //triplicated pipe to tripXsquare filter
	triplicated.To(logxcub)     //triplicated pipe to logxcub filter
	squared.To(tripXsquare)     //squared pipe to tripXsquare filter
	squared.To(logxcub)         //squared pipe to logxcub filter
	cubed.To(substract)         //cubed pipe to substract filter
	loged.To(substract)         //loged pipe to substract filter
	tripXsquared.To(substract)  //tripXsquared pipe to substract filter

	output.To(nil) //final pipe to nil (non filter, it's used to receive data in the last output)

	//6th - Compile every filter for finding errors
	if err := duplicate.Compile(); err != nil {
		panic(err)
	}
	if err := triplicate.Compile(); err != nil {
		panic(err)
	}
	if err := square.Compile(); err != nil {
		panic(err)
	}
	if err := tripXsquare.Compile(); err != nil {
		panic(err)
	}
	if err := logxcub.Compile(); err != nil {
		panic(err)
	}
	if err := substract.Compile(); err != nil {
		panic(err)
	}

	//7th - Set the same signal to every filter
	signal := arch.NewSignal()    //Create signal
	duplicate.SetSignal(signal)   //Signal to duplicate filter
	triplicate.SetSignal(signal)  //Signal to triplicate filter
	square.SetSignal(signal)      //Signal to squre filter
	tripXsquare.SetSignal(signal) //Signal to tripXsquare filter
	logxcub.SetSignal(signal)     //Signal to logxcub filter
	substract.SetSignal(signal)   //Signal to substract filter

	//8th - Run every filter in goruntine
	go duplicate.Run()
	go triplicate.Run()
	go square.Run()
	go tripXsquare.Run()
	go logxcub.Run()
	go substract.Run()

	//9th - Create a gorutine to receive filter data
	//It's posible create first send data to filter and after receive data
	//This gorutine could not be necesary
	//Model example below follows this paradigm.
	count := 10
	go func() {
		for i := 0; i < count; i++ {
			dup := 2 * i
			trip := 3 * dup
			sqrt := math.Sqrt(float64(dup))
			tripXsqrt := float64(trip) * sqrt
			cube := math.Pow(float64(trip)*sqrt, 3)
			log := math.Log(float64(trip) * sqrt)
			//Receive output
			re := output.Get(nil)
			te := cube - log - tripXsqrt
			if te == re {
				fmt.Println(te, " == ", re)
			} else {
				fmt.Println(te, " != ", re)
			}
		}
		signal.Stop()
	}()
	go func() {
		//Send data to filter input for processing
		for i := 0; i < count; i++ {
			input.Set(i)
		}
	}()
	//Wait for every filter stop
	signal.Wait()
	//Print if there is an error
	all := arch.WithFilters(duplicate, triplicate, square, tripXsquare, logxcub, substract)
	for i := range all {
		all[i].PrintErrs()
	}
}

```

#### 2- Create a basic model.

![Example of pipes and filter architecture](https://github.com/stellviaproject/pipfil-arch/tree/master/example/basic-model/basic.png)

```golang
package main

import (
	"fmt"

	arch "github.com/stellviaproject/pipfil-arch"
)

func main() {
	//1st - Create pipes
	input := arch.NewPipe("input", int(0), 1)
	items := arch.NewPipe("items", int(0), 10)
	dupls := arch.NewPipe("dupls", int(0), 10)
	final := arch.NewPipe("final", []int{}, 10)

	//2nd - Create filters
	inc := arch.NewFilterWithPipes("inc", func(input int) []int {
		items := []int{}
		for i := 0; i < input; i++ {
			items = append(items, i)
		}
		return items
	},
		arch.WithPipes(input), //Pipes inputs
		arch.WithPipes(items), //Pipes outputs
		arch.WithLens(),       //Pipes lengths
	)
	dup := arch.NewFilterWithPipes("dup", func(item int) int {
		return item * 2
	},
		arch.WithPipes(items), //Pipes inputs
		arch.WithPipes(dupls), //Pipes outputs
		arch.WithLens(),       //Pipes lengths
	)
	joi := arch.NewFilterWithPipes("joi", func(items []int) []int {
		return items
	},
		arch.WithPipes(dupls),                    //Pipes inputs
		arch.WithPipes(final),                    //Pipes outputs
		arch.WithLens(arch.NewLen(dupls, items)), //Pipes lengths
	)

	//3rd - Create model
	model := arch.NewModel(arch.WithFilters(inc, dup, joi), arch.WithPipes(input), arch.WithPipes(final))
	//4th - Run model
	model.Run()
	//5th - Call model
	slice := model.Call(arch.WithInput(10))[0].([]int)
	fmt.Println(slice) //Print result
	//6th - Stop model
	model.Stop()
	//7th - Test model errors
	if model.HasErrs() {
		model.PrintErrs()
	}
}

```
#### 3- Creating a model with more complexity and with some error.

![Example of pipes and filter architecture](https://github.com/stellviaproject/pipfil-arch/tree/master/example/complexity-plus/complexity-plus.png)

```golang
package main

import (
	"fmt"

	arch "github.com/stellviaproject/pipfil-arch"
)

func main() {
	//1st - Declaring structs for joiners results
	type DupResult struct {
		DupLs []int
		SeqLs []int
	}
	type Pow struct {
		IncLs []*DupResult
		Ls    []int
		PowLs []int
	}
	//2nd - Creating pipes
	inp := arch.NewPipe("input", int(0), 1)
	pow := arch.NewPipe("pow", int(0), 1)
	inc := arch.NewPipe("inc", int(0), 1)
	dup := arch.NewPipe("dup", int(0), 1)
	jinc := arch.NewPipe("jinc", &DupResult{}, 1)
	output := arch.NewPipe("output", &Pow{}, 1)

	//3rd - Creating filters
	powSequencer := arch.NewFilterWithPipes("PowSequencer", func(input int) []int {
		seq := make([]int, input)
		if input == 15 {
			panic(fmt.Errorf("some error"))
		}
		for i := 0; i < input; i++ {
			seq[i] = i * i
		}
		return seq
	},
		arch.WithPipes(inp),
		arch.WithPipes(pow),
		arch.WithLens(),
	)

	incSequencer := arch.NewFilterWithPipes("IncSequencer", func(pow int) []int {
		seq := make([]int, pow)
		for i := 0; i < pow; i++ {
			seq[i] = i
		}
		return seq
	},
		arch.WithPipes(pow),
		arch.WithPipes(inc),
		arch.WithLens(),
	)

	duplicater := arch.NewFilterWithPipes("Duplicater", func(inc int) int {
		return inc * 2
	},
		arch.WithPipes(inc),
		arch.WithPipes(dup),
		arch.WithLens(),
	)

	joinerInc := arch.NewFilterWithPipes("JoinerInc", func(dups []int, seqs []int) *DupResult {
		return &DupResult{
			DupLs: dups,
			SeqLs: seqs,
		}
	},
		arch.WithPipes(dup, inc),
		arch.WithPipes(jinc),
		arch.WithLens(
			arch.NewLen(dup, inc),
			arch.NewLen(inc, inc),
		),
	)

	joinerPow := arch.NewFilterWithPipes("JoinerPow", func(incs []*DupResult, pows []int) *Pow {
		return &Pow{
			IncLs: incs,
			PowLs: pows,
		}
	},
		arch.WithPipes(jinc, pow),
		arch.WithPipes(output),
		arch.WithLens(
			arch.NewLen(jinc, pow),
			arch.NewLen(pow, pow),
		),
	)

	model := arch.NewModel(
		arch.WithFilters(
			powSequencer,
			incSequencer,
			duplicater,
			joinerInc,
			joinerPow,
		),
		arch.WithPipes(inp),
		arch.WithPipes(output),
	)
	//4th - Running model
	model.Run()
	//5th - Calling model
	for i := 10; i < 20; i++ {
		result := model.Call(arch.WithInput(i))[0].(*Pow)
		fmt.Println(result)
		//Testing model error
		if model.HasErrs() {
			model.PrintErrs()
			model.Clear()
		}
	}
	//6th - Stoping model
	model.Stop()
}

```