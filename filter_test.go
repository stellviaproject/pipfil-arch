package arch

import (
	"fmt"
	"math"
	"testing"
)

func TestSlice(t *testing.T) {
	input := NewPipe("input", int(0), 1)
	items := NewPipe("items", int(0), 10)
	dupls := NewPipe("dupls", int(0), 10)
	final := NewPipe("final", []int{}, 10)

	inc := NewFilterWithPipes("inc", func(input int) []int {
		items := []int{}
		for i := 0; i < input; i++ {
			items = append(items, i)
		}
		return items
	},
		WithPipes(input),
		WithPipes(items),
		WithLens(),
	)
	dup := NewFilterWithPipes("dup", func(item int) int {
		return item * 2
	},
		WithPipes(items),
		WithPipes(dupls),
		WithLens(),
	)
	joi := NewFilterWithPipes("joi", func(items []int) []int {
		return items
	},
		WithPipes(dupls),
		WithPipes(final),
		WithLens(NewLen(dupls, items)),
	)

	model := NewModel(WithFilters(inc, dup, joi), WithPipes(input), WithPipes(final))
	model.Run()
	slice := model.Call(WithInput(10))[0].([]int)
	fmt.Println(slice)
	model.Stop()
}

func TestQuick(t *testing.T) {
	input := NewPipe("input", int(0), 1)
	duplicated := NewPipe("duplicated", int(0), 1)
	triplicated := NewPipe("triplicated", int(0), 1)
	squared := NewPipe("squared", float64(0), 1)
	cubed := NewPipe("cubed", float64(0), 1)
	loged := NewPipe("loged", float64(0), 1)
	tripXsquared := NewPipe("3xsqrt", float64(0), 1)
	final := NewPipe("final", float64(0), 1)

	duplicate := NewFilterWithPipes("duplicate", func(input int) int {
		return 2 * input
	},
		WithPipes(input),
		WithPipes(duplicated),
		WithLens(),
	)

	triplicate := NewFilterWithPipes("triplicate", func(input int) int {
		return 3 * input
	},
		WithPipes(duplicated),
		WithPipes(triplicated),
		WithLens(),
	)

	square := NewFilterWithPipes("square", func(input int) float64 {
		return math.Sqrt(float64(input))
	},
		WithPipes(duplicated),
		WithPipes(squared),
		WithLens(),
	)

	tripXsqure := NewFilterWithPipes("3xsqrt", func(trip int, sqrt float64) float64 {
		return float64(trip) * sqrt
	},
		WithPipes(triplicated, squared),
		WithPipes(tripXsquared),
		WithLens(),
	)

	logxcub := NewFilterWithPipes("logxcube", func(trip int, sqrt float64) (float64, float64) {
		return math.Pow(float64(trip)*sqrt, 3), math.Log(float64(trip) * sqrt)
	},
		WithPipes(triplicated, squared),
		WithPipes(cubed, loged),
		WithLens(),
	)

	substract := NewFilterWithPipes("substract", func(cub, log, tripxsqrt float64) float64 {
		return cub - log - tripxsqrt
	},
		WithPipes(cubed, loged, tripXsquared),
		WithPipes(final),
		WithLens(),
	)

	model := NewModel(WithFilters(duplicate, triplicate, square, tripXsqure, logxcub, substract), WithPipes(input), WithPipes(final))
	model.Run()
	for i := 0; i < 10; i++ {
		dup := 2 * i
		trip := 3 * dup
		sqrt := math.Sqrt(float64(dup))
		tripXsqrt := float64(trip) * sqrt
		cube := math.Pow(float64(trip)*sqrt, 3)
		log := math.Log(float64(trip) * sqrt)
		value := model.Call(WithInput(i))[0].(float64)
		te := cube - log - tripXsqrt
		if te == value {
			fmt.Println(te, " == ", value)
		} else {
			fmt.Println(te, " != ", value)
			t.Fail()
		}
	}
	model.Stop()
}

func TestFilter(t *testing.T) {
	input := NewPipe("input", int(0), 1)
	duplicated := NewPipe("duplicated", int(0), 1)
	triplicated := NewPipe("triplicated", int(0), 1)
	squared := NewPipe("squared", float64(0), 1)
	cubed := NewPipe("cubed", float64(0), 1)
	loged := NewPipe("loged", float64(0), 1)
	tripXsquared := NewPipe("3xsqrt", float64(0), 1)
	final := NewPipe("final", float64(0), 1)

	duplicate := NewFilter("duplicate")
	triplicate := NewFilter("triplicate")
	square := NewFilter("square")
	tripXsquare := NewFilter("3xsquare")
	logxcub := NewFilter("multiple")
	substract := NewFilter("substract")

	duplicate.UseFunc(FuncOf(func(input int) int {
		return 2 * input
	}).In(input).Out(duplicated),
	)

	triplicate.UseFunc(FuncOf(func(input int) int {
		return 3 * input
	}).In(duplicated).Out(triplicated))

	square.UseFunc(FuncOf(func(input int) float64 {
		return math.Sqrt(float64(input))
	}).In(duplicated).Out(squared))

	tripXsquare.UseFunc(FuncOf(func(triplicated int, squared float64) float64 {
		return float64(triplicated) * squared
	}).In(triplicated).In(squared).Out(tripXsquared))

	logxcub.UseFunc(FuncOf(func(triplicated int, squared float64) (float64, float64) {
		return math.Pow(float64(triplicated)*squared, 3), math.Log(float64(triplicated) * squared)
	}).In(triplicated).In(squared).Out(cubed).Out(loged))

	substract.UseFunc(FuncOf(func(cubed, loged, tripxsquare float64) float64 {
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

	signal := NewSignal()

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
				t.Fail()
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
	fmt.Println(signal.Err())
}
