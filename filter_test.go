package arch

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// Deteccion de deadlock:
// 1- Puede pasar cuando:
// hay una tuberia conectada a un filtro como entrada pero
// que no recibe nada nunca porque no es entrada de otro filtro
// o del modelo.
// 2- Puede pasar cuando:
// hay una tuberia conectada a la salida de un filtro pero
// no esta conectada a la entrada de otro filtro o no es
// salida del modelo.
// 3- Puede pasar con las condiciones siguientes:
// La salida de un filtro es un slice
// La tuberia de salida de ese filtro no es un slice
// La tuberia que dice la cantidad de elementos (length)
// esta asociada a otro filtro cuya salida es un slice y
// ella misma no es un slice.
// 4- Puede pasar cuando una tuberia de entrada del modelo no
// esta conectada a ningun filtro (de paso chequea la conexion
// de las de salida).
// 5- hay que asegurar que si dos filtros envian elementos po
// una misma tuberia ninguno de los dos hace uso de length.

func TestParallel(t *testing.T) {
	in := NewPipe("in", int(0), 5)
	out := NewPipe("out", int(0), 5)
	ftr := NewFilterWithPipes(
		"for",
		func(c int) int {
			s := 0
			for i := 0; i < c; i++ {
				s += i
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(30)))
			}
			return s
		},
		WithPipes(in),
		WithPipes(out),
		WithLens(),
	)
	model := NewModel(WithFilters(ftr), WithPipes(in), WithPipes(out))
	model.Run()
	const LN = 40
	sums := make([]int, LN)
	wg := sync.WaitGroup{}
	p := make(chan int, 10)
	for i := 0; i < LN; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p <- 0
			sums[i] = model.Call(WithInput(i))[0].(int)
			<-p
		}(i)
	}
	wg.Wait()
	test := make([]int, LN)
	for i := 0; i < LN; i++ {
		sum := 0
		for j := 0; j < i; j++ {
			sum += j
		}
		test[i] = sum
	}
	for i := 0; i < LN; i++ {
		if sums[i] != test[i] {
			t.FailNow()
		}
	}
}

func TestJoiners(t *testing.T) {
	type DupResult struct {
		DupLs []int
		SeqLs []int
	}
	type Pow struct {
		IncLs []*DupResult
		Ls    []int
		PowLs []int
	}
	inp := NewPipe("input", int(0), 1)
	pow := NewPipe("pow", int(0), 1)
	inc := NewPipe("inc", int(0), 1)
	dup := NewPipe("dup", int(0), 1)
	jinc := NewPipe("jinc", &DupResult{}, 1)
	out := NewPipe("out", &Pow{}, 1)

	powSequencer := NewFilterWithPipes("PowSequencer", func(input int) []int {
		seq := make([]int, input)
		for i := 0; i < input; i++ {
			seq[i] = i * i
		}
		return seq
	},
		WithPipes(inp),
		WithPipes(pow),
		WithLens(),
	)

	// duplicater := NewFilterWithPipes("Duplicater", func(inc int) int {
	// 	return inc * 2
	// },
	// 	WithPipes(pow),
	// 	WithPipes(dup),
	// 	WithLens(),
	// )

	// joinerPow := NewFilterWithPipes("JoinerPow", func(dups []int, pows []int) *Pow {
	// 	return &Pow{
	// 		Ls:    dups,
	// 		PowLs: pows,
	// 	}
	// },
	// 	WithPipes(dup, pow),
	// 	WithPipes(out),
	// 	WithLens(
	// 		NewLen(dup, pow),
	// 		NewLen(pow, pow),
	// 	),
	// )

	// model := NewModel(
	// 	WithFilters(
	// 		powSequencer,
	// 		// incSequencer,
	// 		duplicater,
	// 		// joinerInc,
	// 		joinerPow,
	// 	),
	// 	WithPipes(inp),
	// 	WithPipes(out),
	// )
	incSequencer := NewFilterWithPipes("IncSequencer", func(pow int) []int {
		seq := make([]int, pow)
		for i := 0; i < pow; i++ {
			seq[i] = i
		}
		return seq
	},
		WithPipes(pow),
		WithPipes(inc),
		WithLens(),
	)

	duplicater := NewFilterWithPipes("Duplicater", func(inc int) int {
		return inc * 2
	},
		WithPipes(inc),
		WithPipes(dup),
		WithLens(),
	)

	joinerInc := NewFilterWithPipes("JoinerInc", func(dups []int, seqs []int) *DupResult {
		fmt.Println("JoinerInc: ", dups, " ", seqs)
		return &DupResult{
			DupLs: dups,
			SeqLs: seqs,
		}
	},
		WithPipes(dup, inc),
		WithPipes(jinc),
		WithLens(
			NewLen(dup, inc),
			NewLen(inc, inc),
		),
	)

	joinerPow := NewFilterWithPipes("JoinerPow", func(incs []*DupResult, pows []int) *Pow {
		fmt.Println("JoinerPow: ", len(incs), " ", pows)
		return &Pow{
			IncLs: incs,
			PowLs: pows,
		}
	},
		WithPipes(jinc, pow),
		WithPipes(out),
		WithLens(
			NewLen(jinc, pow),
			NewLen(pow, pow),
		),
	)

	model := NewModel(
		WithFilters(
			powSequencer,
			incSequencer,
			duplicater,
			joinerInc,
			joinerPow,
		),
		WithPipes(inp),
		WithPipes(out),
	)
	model.SetParallel(10)
	model.Run()
	result := model.Call(WithInput(10))[0].(*Pow)
	fmt.Println(result)
	model.Stop()
}

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
	all := WithFilters(duplicate, triplicate, square, tripXsquare, logxcub, substract)
	for i := range all {
		all[i].PrintErrs()
	}
}
