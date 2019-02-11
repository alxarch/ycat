package ycat_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/alxarch/ycat"
)

func fullTest(t *testing.T) {
	t.Helper()

}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error {
	return nil
}
func TestParseArgs(t *testing.T) {
	type TestCase struct {
		Args     []string
		Help     bool
		Err      bool
		NumTasks int
		Stdin    string
		Output   string
	}
	tcs := []TestCase{
		{nil, false, false, 2, "1", "1\n"},
		{[]string{"testdata/foo.yaml"}, false, false, 2, "", "foo: bar\n"},
		{[]string{"testdata/foo.yaml", "-e", `x + {bar: "baz"}`}, false, false, 2, "", "foo: bar\nbar: baz\n"},
		// {[]string{""}, false, false, 2, "1", "1\n"},
	}
	for i, tc := range tcs {
		buf := &bytes.Buffer{}
		stdout := &nopCloser{buf}
		stdin := strings.NewReader(tc.Stdin)
		tasks, help, err := ycat.ParseArgs(tc.Args, stdin, stdout)
		if err != nil {
			t.Fatal(i, err)
		}
		if len(tasks) != tc.NumTasks {
			t.Errorf("%s Invalid tasks %d != %d", tc.Args, len(tasks), tc.NumTasks)
		}
		if help != tc.Help {
			t.Errorf("Invalid help")
		}

		p := ycat.MakePipeline(context.Background(), tasks...)
		for err := range p.Errors() {
			if err != nil {
				t.Error(err)
			}
		}
		if buf.String() != tc.Output {
			t.Errorf("%#v Wrong output: %q != %q", tc.Args, buf.String(), tc.Output)
		}
	}

}
