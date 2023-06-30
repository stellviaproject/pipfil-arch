package arch

import (
	"errors"
	"reflect"
)

var (
	ErrPipeRegistered = errors.New("pipe registered")
	ErrPipeNotFound   = errors.New("pipe not found")
	ErrPipeWithLength = errors.New("pipe with length")
)

type PipeCollection interface {
	Set(pipe Pipe) error
	SetNamed(pipe Pipe) error
	Get(pipeType reflect.Type) (Pipe, error)
	GetNamed(name string) (Pipe, error)
	ForEach(action func(Pipe) bool) bool
	GetLenFor(pipe Pipe) (Pipe, error)
	SetLenFor(pipe, length Pipe) error
	Has(pipeType reflect.Type) bool
	HasNamed(name string) bool
	IsOpen() bool
	Close()
}

type collection struct {
	pipes  map[reflect.Type]Pipe
	named  map[string]Pipe
	length map[Pipe]Pipe
}

func newCollection() *collection {
	return &collection{
		pipes:  make(map[reflect.Type]Pipe),
		named:  make(map[string]Pipe),
		length: make(map[Pipe]Pipe),
	}
}

func (coll *collection) Set(pipe Pipe) error {
	if _, ok := coll.pipes[pipe.CheckType()]; ok {
		return ErrPipeRegistered
	}
	coll.pipes[pipe.CheckType()] = pipe
	return nil
}

func (coll *collection) SetNamed(pipe Pipe) error {
	if _, ok := coll.named[pipe.Name()]; ok {
		return ErrPipeRegistered
	}
	coll.named[pipe.Name()] = pipe
	return nil
}

func (coll *collection) Get(pipeType reflect.Type) (Pipe, error) {
	if pipe, ok := coll.pipes[pipeType]; ok {
		return pipe, nil
	}
	return nil, ErrPipeNotFound
}

func (coll *collection) GetNamed(name string) (Pipe, error) {
	if pipe, ok := coll.named[name]; ok {
		return pipe, nil
	}
	return nil, ErrPipeNotFound
}

func (coll *collection) GetLenFor(pipe Pipe) (Pipe, error) {
	if pipe, ok := coll.length[pipe]; ok {
		return pipe, nil
	}
	return nil, ErrPipeNotFound
}

func (coll *collection) SetLenFor(pipe Pipe, length Pipe) error {
	if _, ok := coll.length[pipe]; ok {
		return ErrPipeWithLength
	}
	coll.length[pipe] = length
	return nil
}

func (coll *collection) Has(pipeType reflect.Type) bool {
	_, ok := coll.pipes[pipeType]
	return ok
}

func (coll *collection) HasNamed(name string) bool {
	_, ok := coll.named[name]
	return ok
}

func (coll *collection) IsOpen() bool {
	for _, pipe := range coll.pipes {
		if !pipe.IsOpen() {
			return false
		}
	}
	for _, pipe := range coll.named {
		if !pipe.IsOpen() {
			return false
		}
	}
	return true
}

func (coll *collection) Close() {
	for _, pipe := range coll.pipes {
		pipe.Close()
	}
	for _, pipe := range coll.named {
		pipe.Close()
	}
}

func (coll *collection) ForEach(action func(pipe Pipe) bool) bool {
	for _, pipe := range coll.pipes {
		if !action(pipe) {
			return false
		}
	}
	for _, pipe := range coll.named {
		if !action(pipe) {
			return false
		}
	}
	return false
}
