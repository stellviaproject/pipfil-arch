package arch

import "fmt"

type queue struct {
	inputs          []any
	outputs         map[any]any
	lock, prll, sgn chan int
}

func NewQueue(parallel int) *queue {
	return &queue{
		inputs:  make([]any, 0, parallel),
		outputs: make(map[any]any, parallel),
		lock:    make(chan int, 1),
		prll:    make(chan int, parallel),
		sgn:     make(chan int, 1),
	}
}

// TODO: Probar paralelismo de procesar multiples entradas
func (q *queue) push(item any) {
	q.prll <- 0
	q.lock <- 0
	q.inputs = append(q.inputs, item)
	<-q.lock
}

func (q *queue) set(input, output any) {
	q.lock <- 0
	q.outputs[input] = output
	<-q.lock
	//---------------------------------
	//Use for debuging
	in++
	if len(q.sgn) == cap(q.sgn) {
		fmt.Println("lock sgn<-0 ", in)
	} else {
		fmt.Println("sgn<-0 ", in)
	}
	//---------------------------------
	q.sgn <- 0
	<-q.prll
}

var in int

func (q *queue) pop() any {
	for {
		//--------------------------------
		//Use for debuging
		in--
		if len(q.sgn) == 0 {
			fmt.Println("lock <-sgn ", in)
		} else {
			fmt.Println("<-sgn ", in)
		}
		//--------------------------------
		<-q.sgn
		q.lock <- 0
		input := q.inputs[0]
		if output, ok := q.outputs[input]; ok {
			q.inputs = q.inputs[1:]
			delete(q.outputs, input)
			<-q.lock
			q.sgn <- 0 //signal for next item
			return output
		}
		<-q.lock
	}
	//return nil
}
