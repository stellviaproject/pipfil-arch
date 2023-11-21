package arch

type queue struct {
	parallel        int
	inputs          []any
	outputs         []chan any
	lock, prll, sgn chan int
}

func newQueue(parallel int) *queue {
	return &queue{
		parallel: parallel,
		inputs:   make([]any, 0, parallel),
		outputs:  make([]chan any, 0, parallel),
		lock:     make(chan int, 1),
		prll:     make(chan int, parallel),
		sgn:      make(chan int, 1),
	}
}

func (q *queue) push(item any) chan any {
	q.prll <- 0
	q.lock <- 0
	q.inputs = append(q.inputs, item)
	output := make(chan any, 1)
	q.outputs = append(q.outputs, output)
	<-q.lock
	return output
}

func (q *queue) set() {
	<-q.prll
}

func (q *queue) exit() {
	q.sgn <- 0
}

func (q *queue) run(fn func(output any)) {
	if q.parallel > 1 {
		go func() {
			exit := false
			for {
				select {
				case <-q.sgn:
					exit = true
				default:
				}
				q.lock <- 0
				if len(q.outputs) > 0 {
					select {
					case r := <-q.outputs[0]:
						<-q.lock
						fn(r)
						q.lock <- 0
						close(q.outputs[0])
						q.inputs = q.inputs[1:]
						q.outputs = q.outputs[1:]
					default:
					}
				}
				if len(q.outputs) == 0 && exit {
					<-q.lock
					break
				}
				<-q.lock
			}
		}()
	}
}
