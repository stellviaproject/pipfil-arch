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
