package arch

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// It's produced when data send to pipe doesn't match with pipe data type in its definition
//var ErrPipeTypeMismatch = errors.New("pipe input data type mismatch")

// It's produced when a filter try to get data from pipe and it's not linked to pipe, use method To(filter Filter)
var ErrUnRegisteredFilter = errors.New("unregistered filter")

// It's produced when a filter is registered and you try to register it again
var ErrFilterRegistered = errors.New("filter registered")

// Represents a pipe for pipes-filters architectures
type Pipe interface {
	Name() string            //Pipe name
	To(filter Filter) error  //Link pipe to filter input
	LenTo(pipe Pipe) error   //Set pipe to send length
	Set(data any)            //Send data to pipe
	Get(filter Filter) any   //Receive data from pipe
	SetLen(len int)          //Send length to all pipes
	Len(pipe Pipe) int       //Get length for pipe
	CheckType() reflect.Type //Pipe data type
	IsOpen() bool            //Test if pipe internal channels are opened
	Close()                  //Close pipe internal channels, filters associated with pipe will be stopped
}

// pipe implementation
type pipe struct {
	name      string
	conn      map[Filter]chan any //pipe data channel
	len       map[Pipe]chan int   //pipe length channel
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
		len:       make(map[Pipe]chan int, 10),   //set length of wrapped
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
func (pipe *pipe) LenTo(p Pipe) error {
	if _, ok := pipe.len[p]; ok {
		return ErrFilterRegistered
	}
	pipe.len[p] = make(chan int, pipe.buffer)
	return nil
}

// Send data through pipe
func (pipe *pipe) Set(data any) {
	pipe.mtx.Lock()
	defer pipe.mtx.Unlock()
	//Get input data data type
	inType := reflect.TypeOf(data)
	//Check input data type
	if inType != nil && !inType.AssignableTo(pipe.checkType) {
		panic(fmt.Errorf("pipe '%s' receive type '%s' but is defined as '%s'", pipe.name, inType, pipe.checkType))
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
func (pipe *pipe) Len(p Pipe) int {
	ch, ok := pipe.len[p]
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
