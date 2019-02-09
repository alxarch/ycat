package ycat

import (
	"context"
)

type WriteStream interface {
	Push(*Value) bool
}
type ReadStream interface {
	Next() (*Value, bool)
}
type Stream interface {
	ReadStream
	WriteStream
}
type StreamTask interface {
	Run(s Stream) error
}

type StreamFunc func(s Stream) error

func (f StreamFunc) Run(s Stream) error { return f(s) }

type Consumer interface {
	Consume(s ReadStream) error
}
type ConsumerFunc func(s ReadStream) error

func (f ConsumerFunc) Consume(s ReadStream) error { return f(s) }
func (f ConsumerFunc) Run(s Stream) error         { return f(s) }

type Producer interface {
	Produce(s WriteStream) error
}
type ProducerFunc func(s WriteStream) error

func (f ProducerFunc) Produce(s WriteStream) error { return f(s) }
func (f ProducerFunc) Run(s Stream) error {
	Drain(s)
	return f(s)
}

type Producers []Producer

// Run implements StreamTask for Producers
func (tasks Producers) Run(s Stream) error { return tasks.Produce(s) }

// Produce implements Producer for Producers
func (tasks Producers) Produce(s WriteStream) error {
	for _, task := range tasks {
		if err := task.Produce(s); err != nil {
			return err
		}
	}
	return nil
}

type stream struct {
	done <-chan struct{}
	src  <-chan *Value
	out  chan<- *Value
}

func (s *stream) Next() (v *Value, ok bool) {
	select {
	case v, ok = <-s.src:
		// println("s.value", ok)
	case <-s.done:
		// println("next s.done", ok)
	}
	return
}

func (s *stream) Push(v *Value) bool {
	select {
	case s.out <- v:
		return true
	case <-s.done:
		return false
	}
}

type NullStream struct{}

func (NullStream) Produce(s WriteStream) error {
	s.Push(&Value{Null, nil})
	return nil
}

type ToArray struct{}

func (ToArray) Init(_ context.Context) int {
	return 0
}

func (ToArray) Run(s Stream) (err error) {
	var arr []*Value
	for {
		v, ok := s.Next()
		if ok {
			arr = append(arr, v)
		} else {
			break
		}
	}
	if arr != nil {
		v := Value{Array, arr}
		s.Push(&v)
	}
	return
}

func Drain(s Stream) bool {
	for {
		v, ok := s.Next()
		if !ok {
			return true
		}
		if !s.Push(v) {
			return false
		}
	}

}

// type DrainTask struct{}

// func (DrainTask) Run(s Stream) error {
// 	Drain(s)
// 	return nil
// }
// func (DrainTask) Init(ctx context.Context) int {
// 	return 0
// }

// type drainStream struct {
// 	Stream
// 	Drained bool
// }

// func DrainStream(s Stream) WriteStream {
// 	return &drainStream{s, false}
// }

// func (s *drainStream) Push(v *Value) bool {
// 	if !s.Drained {
// 		s.Drained = true
// 		if !Drain(s.Stream) {
// 			return false
// 		}
// 	}
// 	return s.Stream.Push(v)
// }
