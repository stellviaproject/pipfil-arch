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
