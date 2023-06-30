package arch

import (
	"errors"
	"reflect"
	"sync"
)

// It's produced when data send to pipe doesn't match with pipe data type in its definition
var ErrPipeTypeMismatch = errors.New("pipe input data type mismatch")

// It's produced when a filter try to get data from pipe and it's not linked to pipe, use method To(filter Filter)
var ErrUnRegisteredFilter = errors.New("unregistered filter")

// It's produced when a filter is registered and you try to register it again
var ErrFilterRegistered = errors.New("filter registered")

// Represents a pipe for pipes-filters architectures
type Pipe interface {
	Name() string           //Pipe name
	To(filter Filter) error //Link pipe to filter input
	LenTo(filter Filter) error
	Set(data any)          //Send data to pipe
	Get(filter Filter) any //Receive data from pipe
	SetLen(len int)
	Len(filter Filter) int
	CheckType() reflect.Type //Pipe data type
	IsOpen() bool            //Test if pipe internal channels are opened
	Close()                  //Close pipe internal channels, filters associated with pipe will be stopped
	//IsWrapp() bool           //Tell if pipe is wrapped (Example: sending []int will send int one by one)
}

// pipe implementation
type pipe struct {
	name      string
	conn      map[Filter]chan any //pipe data channel
	len       map[Filter]chan int //pipe length channel
	buffer    int
	checkType reflect.Type
	isOpen    bool
	mtx       sync.Mutex
}

// Create a new pipe with checkType and buffer size
func NewPipe(name string, checkType any, buffer int) Pipe {
	pipeType := reflect.TypeOf(checkType)
	if pipeType.Kind() == reflect.Ptr && pipeType.Elem().Kind() == reflect.Interface {
		pipeType = pipeType.Elem()
	}
	return &pipe{
		name:      name,
		checkType: pipeType,                      //set check type
		conn:      make(map[Filter]chan any, 10), //set pipe buffer
		len:       make(map[Filter]chan int, 10), //set length of wrapped
		buffer:    buffer,
		isOpen:    true,
	}
}

func (pipe *pipe) Name() string {
	return pipe.name
}

// Link pipe to filter input
func (pipe *pipe) To(filter Filter) error {
	if _, ok := pipe.conn[filter]; ok {
		return ErrFilterRegistered
	}
	pipe.conn[filter] = make(chan any, pipe.buffer)
	return nil
}

// Link pipe length to filter input
func (pipe *pipe) LenTo(filter Filter) error {
	if _, ok := pipe.len[filter]; ok {
		return ErrFilterRegistered
	}
	pipe.len[filter] = make(chan int, pipe.buffer)
	return nil
}

// Send data through pipe
func (pipe *pipe) Set(data any) {
	pipe.mtx.Lock()
	defer pipe.mtx.Unlock()
	//Get input data data type
	inType := reflect.TypeOf(data)
	//Check input data type
	if inType != pipe.checkType {
		panic(ErrPipeTypeMismatch)
	}
	//Make sure every channel is receiving data without lost it
	wg := sync.WaitGroup{}
	for _, ch := range pipe.conn {
		wg.Add(1)
		go func(ch chan any) {
			ch <- data
			wg.Done()
		}(ch)
	}
	wg.Wait()
}

// Get data from pipe
func (pipe *pipe) Get(filter Filter) any {
	ch, ok := pipe.conn[filter]
	if !ok {
		panic(ErrUnRegisteredFilter)
	}
	return <-ch //take data from channel
}

// Send data through pipe
func (pipe *pipe) SetLen(length int) {
	pipe.mtx.Lock()
	defer pipe.mtx.Unlock()
	//Get input data data type
	//Make sure every channel is receiving data without lost it
	wg := sync.WaitGroup{}
	for _, ch := range pipe.len {
		wg.Add(1)
		go func(ch chan int) {
			ch <- length
			wg.Done()
		}(ch)
	}
	wg.Wait()
}

// Get data from pipe
func (pipe *pipe) Len(filter Filter) int {
	ch, ok := pipe.len[filter]
	if !ok {
		panic(ErrUnRegisteredFilter)
	}
	return <-ch //take data from channel
}

// Get pipe internal checkType
func (pipe *pipe) CheckType() reflect.Type {
	return pipe.checkType
}

func (pipe *pipe) IsOpen() bool {
	return pipe.isOpen
}

func (pipe *pipe) Close() {
	if pipe.isOpen {
		for _, ch := range pipe.conn {
			close(ch)
		}
		pipe.isOpen = false
	}
}
