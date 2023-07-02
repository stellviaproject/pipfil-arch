package main

import (
	"fmt"
	"math"

	arch "github.com/stellviaproject/pipfil-arch"
)

func main() {
	input := arch.NewPipe("input", int(0), 1)
	duplicated := arch.NewPipe("duplicated", int(0), 1)
	triplicated := arch.NewPipe("triplicated", int(0), 1)
	squared := arch.NewPipe("squared", float64(0), 1)
	cubed := arch.NewPipe("cubed", float64(0), 1)
	loged := arch.NewPipe("loged", float64(0), 1)
	tripXsquared := arch.NewPipe("3xsqrt", float64(0), 1)
	final := arch.NewPipe("final", float64(0), 1)

	duplicate := arch.NewFilter("duplicate")
	triplicate := arch.NewFilter("triplicate")
	square := arch.NewFilter("square")
	tripXsquare := arch.NewFilter("3xsquare")
	logxcub := arch.NewFilter("multiple")
	substract := arch.NewFilter("substract")

	duplicate.UseFunc(arch.FuncOf(func(input int) int {
		return 2 * input
	}).In(input).Out(duplicated),
	)

	triplicate.UseFunc(arch.FuncOf(func(input int) int {
		return 3 * input
	}).In(duplicated).Out(triplicated))

	square.UseFunc(arch.FuncOf(func(input int) float64 {
		return math.Sqrt(float64(input))
	}).In(duplicated).Out(squared))

	tripXsquare.UseFunc(arch.FuncOf(func(triplicated int, squared float64) float64 {
		return float64(triplicated) * squared
	}).In(triplicated).In(squared).Out(tripXsquared))

	logxcub.UseFunc(arch.FuncOf(func(triplicated int, squared float64) (float64, float64) {
		return math.Pow(float64(triplicated)*squared, 3), math.Log(float64(triplicated) * squared)
	}).In(triplicated).In(squared).Out(cubed).Out(loged))

	substract.UseFunc(arch.FuncOf(func(cubed, loged, tripxsquare float64) float64 {
		return cubed - loged - tripxsquare
	}).In(cubed).In(loged).In(tripXsquared).Out(final))

	duplicate.Input().SetNamed(input)
	duplicate.Output().SetNamed(duplicated)

	triplicate.Input().SetNamed(duplicated)
	triplicate.Output().SetNamed(triplicated)

	square.Input().SetNamed(duplicated)
	square.Output().SetNamed(squared)

	tripXsquare.Input().SetNamed(triplicated)
	tripXsquare.Input().SetNamed(squared)
	tripXsquare.Output().SetNamed(tripXsquared)

	logxcub.Input().SetNamed(triplicated)
	logxcub.Input().SetNamed(squared)
	logxcub.Output().SetNamed(cubed)
	logxcub.Output().SetNamed(loged)

	substract.Input().SetNamed(cubed)
	substract.Input().SetNamed(loged)
	substract.Input().SetNamed(tripXsquared)
	substract.Output().SetNamed(final)

	input.To(duplicate)
	duplicated.To(triplicate)
	duplicated.To(square)
	triplicated.To(tripXsquare)
	triplicated.To(logxcub)
	squared.To(tripXsquare)
	squared.To(logxcub)
	cubed.To(substract)
	loged.To(substract)
	tripXsquared.To(substract)

	final.To(nil)

	signal := arch.NewSignal()

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

	duplicate.SetSignal(signal)
	triplicate.SetSignal(signal)
	square.SetSignal(signal)
	tripXsquare.SetSignal(signal)
	logxcub.SetSignal(signal)
	substract.SetSignal(signal)

	go duplicate.Run()
	go triplicate.Run()
	go square.Run()
	go tripXsquare.Run()
	go logxcub.Run()
	go substract.Run()

	count := 10
	go func() {
		for i := 0; i < count; i++ {
			dup := 2 * i
			trip := 3 * dup
			sqrt := math.Sqrt(float64(dup))
			tripXsqrt := float64(trip) * sqrt
			cube := math.Pow(float64(trip)*sqrt, 3)
			log := math.Log(float64(trip) * sqrt)
			re := final.Get(nil)
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
		for i := 0; i < count; i++ {
			input.Set(i)
		}
	}()
	signal.Wait()
	all := arch.WithFilters(duplicate, triplicate, square, tripXsquare, logxcub, substract)
	for i := range all {
		all[i].PrintErrs()
	}
}
