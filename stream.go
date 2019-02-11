package ycat

import (
	"log"
	"os"
)

// WriteStream is a writtable stream of values
type WriteStream interface {
	Push(RawValue) bool
}

// ReadStream is a readable stream of values
type ReadStream interface {
	Next() (RawValue, bool)
}

// Stream is a readable/writable stream of values
type Stream interface {
	ReadStream
	WriteStream
}

//StreamTask represents a task to run on a Stream
type StreamTask interface {
	Run(s Stream) error
}

// StreamFunc is a StreamTask callback
type StreamFunc func(s Stream) error

//Run implements StreamTask for StramFunc
func (f StreamFunc) Run(s Stream) error { return f(s) }

// Consumer consumes values from a readable stream
type Consumer interface {
	Consume(s ReadStream) error
}

// ConsumerFunc is a Consumer callback
type ConsumerFunc func(s ReadStream) error

// Consume implements Consumer
func (f ConsumerFunc) Consume(s ReadStream) error { return f(s) }

// Run implements StreamTask
func (f ConsumerFunc) Run(s Stream) error { return f(s) }

// Producer generates values for a WriteStream
type Producer interface {
	Produce(s WriteStream) error
}

// ProducerFunc is a Producer callback
type ProducerFunc func(s WriteStream) error

// Produce implements Producer for ProducerFunc
func (f ProducerFunc) Produce(s WriteStream) error { return f(s) }

// Run implements StreamTask for ProducerFunc
func (f ProducerFunc) Run(s Stream) error {
	if Drain(s) {
		return f(s)
	}
	return nil
}

// Producers is a sequence of Producers
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
	src  <-chan RawValue
	out  chan<- RawValue
}

// Next implements ReadStream
func (s *stream) Next() (v RawValue, ok bool) {
	select {
	case v, ok = <-s.src:
		// println("s.value", ok)
	case <-s.done:
		// println("next s.done", ok)
	}
	return
}

// Push implements WriteStream
func (s *stream) Push(v RawValue) bool {
	select {
	case s.out <- v:
		return true
	case <-s.done:
		return false
	}
}

// NullStream is a Producer that pushes a null
type NullStream struct{}

// Produce implements Producer for NullStream
func (NullStream) Produce(s WriteStream) error {
	s.Push("null")
	return nil
}

type Debug string

func (d Debug) Run(s Stream) (err error) {
	logger := log.New(os.Stderr, string(d), 0)
	for {
		v, ok := s.Next()
		if !ok {
			logger.Println("EOF")
			return nil
		}
		logger.Println("Value", v)
		if !s.Push(v) {
			logger.Println("Push end")
			return nil
		}
	}
}

// ToArray concatenates stream values to an array
type ToArray struct{}

// Run implements StreamTask for ToArray
func (ToArray) Run(s Stream) (err error) {
	var values []RawValue
	for {
		v, ok := s.Next()
		if ok {
			values = append(values, v)
		} else {
			break
		}
	}
	if values != nil {
		s.Push(RawValueArray(values...))
	}
	return
}

// Drain is a helper that drains all values from a stream
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

// func (s *drainStream) Push(v RawValue) bool {
// 	if !s.Drained {
// 		s.Drained = true
// 		if !Drain(s.Stream) {
// 			return false
// 		}
// 	}
// 	return s.Stream.Push(v)
// }
