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
	out := arch.NewPipe("out", &Pow{}, 1)

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
		arch.WithPipes(out),
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
		arch.WithPipes(out),
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
