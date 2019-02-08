package ycat

import (
	"context"
	"io"
	"sync"
)

type Stream interface {
	Next() (*Value, bool)
	Push(*Value) bool
}
type StreamTask interface {
	Run(s Stream) error
	Size(ctx context.Context) int
}

type StreamTaskSequence []StreamTask

func (tasks StreamTaskSequence) Size(ctx context.Context) (size int) {
	for _, task := range tasks {
		if s := task.Size(ctx); s > size {
			size = s
		}
	}
	return
}

func (tasks StreamTaskSequence) Run(s Stream) error {
	for _, task := range tasks {
		if err := task.Run(s); err != nil {
			return err
		}
	}
	return nil
}

type StreamFunc func(s Stream) error

func (f StreamFunc) Run(s Stream) error {
	return f(s)
}

func (f StreamFunc) Size(ctx context.Context) int {
	var task StreamTask = f
	if size, ok := ctx.Value(task).(int); ok {
		return size
	}
	return 0
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

func (p Pipeline) Pipe(ctx context.Context, tasks ...StreamTask) Pipeline {
	ecs := make([]<-chan error, 0, len(tasks)+1)
	ecs = append(ecs, p.Errors())
	for _, t := range tasks {
		p = p.RunTask(ctx, t)
		ecs = append(ecs, p.Errors())
	}
	return Pipeline{p.Values(), MergeErrors(ecs...)}
}
func BlankPipeline() Pipeline {
	out := make(chan *Value)
	close(out)
	errc := make(chan error)
	close(errc)
	return Pipeline{out, errc}
}

func (p Pipeline) RunTask(ctx context.Context, task StreamTask) Pipeline {
	src := p.Values()
	out := make(chan *Value, task.Size(ctx))
	errc := make(chan error, 1)
	s := stream{
		done: ctx.Done(),
		src:  src,
		out:  out,
	}
	go func() {
		defer close(errc)
		defer close(out)
		errc <- task.Run(&s)
	}()
	return Pipeline{out, errc}
}

func ReadFromTask(r io.Reader, format Format) StreamTask {
	return StreamFunc(func(s Stream) error {
		// Needed for reading in the middle of a pipeline
		if err := Drain(s); err != nil {
			return err
		}
		dec := NewDecoder(r, format)
		for {
			v := new(Value)
			if err := dec.Decode(v); err != nil {
				if err == io.EOF {
					// println("read eof")
					return nil
				}
				// println("read err", err.Error())
				return err
			}

			if !s.Push(v) {
				// println("push failed")
				return nil
			}
			// println("push ok")
		}
	})
}

func StreamWriteTo(w io.WriteCloser, format Output) StreamTask {
	return StreamFunc(func(s Stream) error {
		defer w.Close()
		enc := NewEncoder(w, format)
		for {
			v, ok := s.Next()
			if !ok {
				// println("w no next")
				return nil
			}
			// println("next")
			if err := enc.Encode(v); err != nil {
				// println("encode err")
				return err
			}
		}
	})
}

type NullStream struct{}

func (NullStream) Run(s Stream) error {
	s.Push(&Value{Null, nil})
	return nil
}
func (NullStream) Size(_ context.Context) int {
	return 0
}

type ToArray struct{}

func (ToArray) Size(_ context.Context) int {
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

type Pipeline struct {
	values <-chan *Value
	errors <-chan error
}

func (p *Pipeline) Errors() <-chan error {
	if p.errors == nil {
		ch := make(chan error)
		close(ch)
		p.errors = ch
	}
	return p.errors
}
func (p *Pipeline) Values() <-chan *Value {
	if p.values == nil {
		ch := make(chan *Value)
		close(ch)
		p.values = ch
	}
	return p.values
}
func MergeErrors(cs ...<-chan error) <-chan error {
	switch n := len(cs); n {
	case 1:
		return cs[0]
	case 0:
		out := make(chan error)
		close(out)
		return out
	default:
		out := make(chan error, n)
		wg := sync.WaitGroup{}
		wg.Add(n)
		for i := range cs {
			c := cs[i]
			go func() {
				defer wg.Done()
				for v := range c {
					out <- v
				}
			}()
		}
		go func() {
			defer close(out)
			wg.Wait()
		}()
		return out
	}
}

func Drain(s Stream) (err error) {
	for {
		v, ok := s.Next()
		if !ok {
			return
		}
		if !s.Push(v) {
			return
		}
	}
}

// func (tasks StreamTaskSequence) Run(ctx context.Context) (p *Pipeline) {
// 	ecs := make([]<-chan error, 0, len(tasks))
// 	values := make([]<-chan *Value, 0, len(tasks))
// 	src := make(chan *Value)
// 	close(src)
// 	wg := sync.WaitGroup{}
// 	wg.Add(len(tasks))
// 	for _, t := range tasks {
// 		p = RunTask(ctx, t, src)
// 		values = append(values, p.Values())
// 		ecs = append(ecs, p.Errors())
// 	}
// 	out := make(chan *Value)
// 	done := ctx.Done()
// 	go func() {
// 		defer close(out)
// 		for _, src := range values {
// 			for v := range src {
// 				select {
// 				case out <- v:
// 				case <-done:
// 					return
// 				}
// 			}
// 		}
// 	}()

// 	return &Pipeline{out, MergeErrors(ecs...)}
// }
