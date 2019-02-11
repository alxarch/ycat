package ycat_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/alxarch/ycat"
)

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error {
	return nil
}
func TestParseArgs(t *testing.T) {
	type TestCase struct {
		Args   []string
		Stdin  string
		Stdout string
	}
	tcs := []TestCase{
		{[]string{"-", "-e", "x"}, "null\n---\nnull\n", "null\n---\nnull\n"},
		{nil, "1", "1\n"},
		{[]string{"-n"}, "", "null\n"},
		{[]string{"-n", "-v", "y==foo", "-e", "y"}, "", "foo\n"},
		{[]string{"-n", "--input-var", "y", "-e", "y"}, "", "null\n"},
		{[]string{"testdata/foo.yaml", "-i", "foo=testdata/foo.libsonnet", "-e", "foo.hello(x, 'world')"}, "", "foo: bar\nname: world\n"},
		{[]string{"testdata/foo.yaml"}, "", "foo: bar\n"},
		{[]string{"-y", "testdata/foo.yaml"}, "", "foo: bar\n"},
		{[]string{"testdata/foo.yaml", "-o", "j"}, "", `{"foo":"bar"}` + "\n"},
		{[]string{"testdata/foo.yaml", "testdata/bar.json"}, "", "foo: bar\n---\nbar: foo\n"},
		{[]string{"testdata/foo.yaml", "testdata/bar.json", "-a"}, "", "- foo: bar\n- bar: foo\n"},
		{[]string{
			"testdata/foo.yaml",
			"-e", `{bar: "baz"} + x`,
		}, "", "bar: baz\nfoo: bar\n"},
		// {[]string{""}, false, false, 2, "1", "1\n"},
	}
	for i, tc := range tcs {
		name := fmt.Sprintf("%v", tc.Args)
		t.Run(name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			stdout := &nopCloser{buf}
			stdin := strings.NewReader(tc.Stdin)
			tasks, _, err := ycat.ParseArgs(tc.Args, stdin, stdout)
			if err != nil {
				t.Fatal(i, err)
			}
			p := ycat.MakePipeline(context.Background(), tasks...)
			for err := range p.Errors() {
				if err != nil {
					t.Error(err)
				}
			}
			if buf.String() != tc.Stdout {
				t.Errorf("Wrong output: %q != %q", buf.String(), tc.Stdout)
			}
		})
	}

}
