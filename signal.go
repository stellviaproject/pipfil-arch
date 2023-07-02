package arch

import "time"

// Create a Signal
func NewSignal() Signal {
	return &signal{}
}

type signal struct {
	count int
	stop  chan int
}

func (sg *signal) Stop() {
	for i := 0; i < sg.count; i++ {
		sg.stop <- 0
	}
}

func (sg *signal) init() {
	sg.count++
	sg.stop = make(chan int, sg.count)
}

func (sg *signal) Wait() {
	if sg.count == 0 {
		return
	}
	sg.init()
	for {
		select {
		case <-sg.stop:
			return
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func (sg *signal) tryStop() bool {
	if sg.stop == nil {
		return true
	}
	select {
	case <-sg.stop:
		return true
	default:
		return false
	}
}
