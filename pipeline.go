package ycat

import (
	"context"
	"sync"
)

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
func MakePipeline(ctx context.Context, tasks ...StreamTask) (p *Pipeline) {
	p = new(Pipeline)
	return p.Pipe(ctx, tasks...)
}
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
	var out chan *Value
	switch task := task.(type) {
	case Consumer:
		out = make(chan *Value)
		close(out)
		s.out = out
		go func() {
			defer close(errc)
			errc <- task.Consume(&s)
		}()
	case Producer:
		out = make(chan *Value, 1)
		s.out = out
		go func() {
			defer close(errc)
			defer close(out)
			Drain(&s)
			errc <- task.Produce(&s)
		}()
	default:
		out = make(chan *Value)
		s.out = out
		go func() {
			defer close(errc)
			defer close(out)
			errc <- task.Run(&s)
		}()
	}
	return &Pipeline{out, errc}

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
