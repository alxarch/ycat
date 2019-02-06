package ycat

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"sync"
)

type PipelineFunc func(ctx context.Context, in <-chan *Value, out chan<- *Value) error

func BuildPipeline(ctx context.Context, steps ...PipelineFunc) (<-chan *Value, <-chan error) {
	var (
		last = closedIn()
		errs []<-chan error
	)

	for i := range steps {
		out := make(chan *Value)
		errc := make(chan error, 1)
		fn := steps[i]
		src := last
		go func() {
			defer close(errc)
			defer close(out)
			if err := fn(ctx, src, out); err != nil {
				errc <- err
			}
		}()
		last = out
		errs = append(errs, errc)
	}
	return last, MergeErrors(errs...)
}
func closedErrs() <-chan error {
	c := make(chan error)
	close(c)
	return c
}

func WriteTo(w io.WriteCloser, format Format) PipelineFunc {
	enc := NewEncoder(w, format)
	return func(ctx context.Context, in <-chan *Value, out chan<- *Value) error {
		defer w.Close()
		for v := range in {
			if err := enc.Encode(v); err != nil {
				return err
			}
		}
		return nil
	}
}
func closedIn() <-chan *Value {
	ch := make(chan *Value)
	close(ch)
	return ch
}

func Drain(ctx context.Context, src <-chan *Value, out chan<- *Value) error {
	if src == nil {
		return nil
	}
	for v := range src {
		select {
		case out <- v:
		case <-ctx.Done():
			return nil
		}
	}
	return nil
}

// AndThen starts emitting values from fn after all values from in
// This allows generator Pipeline functions to be added in sequence
func AndThen(fn PipelineFunc) PipelineFunc {
	return func(ctx context.Context, in <-chan *Value, out chan<- *Value) error {
		if in == nil {
			in = closedIn()
		} else if err := Drain(ctx, in, out); err != nil {
			return err
		}
		return fn(ctx, in, out)
	}
}

func ReadFiles(files ...InputFile) PipelineFunc {
	return func(ctx context.Context, in <-chan *Value, out chan<- *Value) error {
		for i := range files {
			f := &files[i]
			format := f.Format
			if format == Auto {
				format = DetectFormat(f.Path)
			}

			var r io.ReadCloser
			switch f.Path {
			case "", "-":
				r = ioutil.NopCloser(os.Stdin)
			default:
				f, err := os.Open(f.Path)
				if err != nil {
					return err
				}
				r = f
			}
			if err := ReadFrom(r, format)(ctx, in, out); err != nil {
				return err
			}
		}
		return nil
	}
}

func ReadFrom(r io.ReadCloser, format Format) PipelineFunc {
	dec := NewDecoder(r, format)
	return func(ctx context.Context, _ <-chan *Value, out chan<- *Value) error {
		defer r.Close()
		for {
			v := new(Value)
			if err := dec.Decode(v); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			select {
			case out <- v:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func ToArray(ctx context.Context, in <-chan *Value, out chan<- *Value) error {
	var arr []*Value
	for v := range in {
		arr = append(arr, v)
	}
	if arr != nil {
		select {
		case out <- &Value{
			Type:  Array,
			Value: arr,
		}:
		case <-ctx.Done():
			return nil
		}
	}
	return nil
}

func MergeErrors(errcs ...<-chan error) <-chan error {
	out := make(chan error, len(errcs))
	wg := sync.WaitGroup{}
	wg.Add(len(errcs))
	for i := range errcs {
		errc := errcs[i]
		go func() {
			defer wg.Done()
			for err := range errc {
				out <- err
			}
		}()
	}
	go func() {
		defer close(out)
		wg.Wait()
	}()
	return out
}

func withCancel(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithCancel(ctx)
}

func wrapErr(err error) <-chan error {
	c := make(chan error, 1)
	c <- err
	close(c)
	return c
}
