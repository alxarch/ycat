package ycat

import (
	"context"
	"sync"
)

// Pipeline is the endpoint of a value stream process
type Pipeline struct {
	values <-chan RawValue
	errors <-chan error
}

// Errors returns a channel with errors from tasks
func (p *Pipeline) Errors() <-chan error {
	if p.errors == nil {
		ch := make(chan error)
		close(ch)
		p.errors = ch
	}
	return p.errors
}

// Values returns a channel with values from tasks
func (p *Pipeline) Values() <-chan RawValue {
	if p.values == nil {
		ch := make(chan RawValue)
		close(ch)
		p.values = ch
	}
	return p.values
}

// MakePipeline builds and runs a pipeline
func MakePipeline(ctx context.Context, tasks ...StreamTask) (p *Pipeline) {
	p = new(Pipeline)
	return p.Pipe(ctx, tasks...)
}

// Pipe adds tasks ro a pipeline
func (p *Pipeline) Pipe(ctx context.Context, tasks ...StreamTask) *Pipeline {
	ecs := make([]<-chan error, 0, len(tasks)+1)
	ecs = append(ecs, p.Errors())
	for _, t := range tasks {
		p = p.task(ctx, t)
		ecs = append(ecs, p.Errors())
	}
	return &Pipeline{p.Values(), MergeErrors(ecs...)}
}

func (p *Pipeline) task(ctx context.Context, task StreamTask) *Pipeline {
	src := p.Values()
	errc := make(chan error, 1)
	s := stream{
		done: ctx.Done(),
		src:  src,
	}
	var out chan RawValue
	switch task := task.(type) {
	case Consumer:
		out = make(chan RawValue)
		close(out)
		s.out = out
		go func() {
			defer close(errc)
			errc <- task.Consume(&s)
			// Drain src
			for _ = range src {
			}
		}()
	case Producer:
		out = make(chan RawValue, 1)
		s.out = out
		go func() {
			defer close(errc)
			defer close(out)
			Drain(&s)
			errc <- task.Produce(&s)
		}()
	default:
		out = make(chan RawValue)
		s.out = out
		go func() {
			defer close(errc)
			defer close(out)
			errc <- task.Run(&s)
			// Drain src
			for _ = range src {
			}
		}()
	}
	return &Pipeline{out, errc}

}

// MergeErrors is a helper function that merges error channels
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
