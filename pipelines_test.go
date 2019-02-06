package ycat_test

import (
	"context"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/alxarch/ycat"
)

func TestReadFrom(t *testing.T) {
	src := `
foo: bar
---
bar: baz
`
	r := strings.NewReader(src)
	ctx := context.Background()
	read := ycat.ReadFrom(ioutil.NopCloser(r), ycat.YAML)
	out := make(chan *ycat.Value, 2)
	err := read(ctx, nil, out)
	if err != nil {
		t.Fatal(err)
	}

	{
		r.Reset(src)
		read := ycat.ReadFrom(ioutil.NopCloser(r), ycat.YAML)
		out, errs := ycat.BuildPipeline(ctx, read)
		if v := <-out; v.Type != ycat.Object {
			t.Errorf("Invalid value type: %s", v.Type)
		}
		if v := <-out; v.Type != ycat.Object {
			t.Errorf("Invalid value type: %s", v.Type)
		} else if obj, ok := v.Value.(map[string]interface{}); !ok {
			t.Errorf("Invalid value type: %s", v.Type)
		} else if obj["bar"] != "baz" {
			t.Errorf("Invalid value: %v", obj)
		}
		for e := range errs {
			if e != nil {
				t.Error(e)
			}
		}

	}

}
