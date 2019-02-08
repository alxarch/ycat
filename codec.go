package ycat

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// Format is input file format
type Format uint

const (
	Auto Format = iota
	YAML
	JSON
)

func FormatFromString(s string) Format {
	switch strings.ToLower(s) {
	case "json", "j":
		return JSON
	case "yaml", "y":
		return YAML
	default:
		return Auto
	}
}

func DetectFormat(path string) Format {
	if strings.HasSuffix(path, ".json") {
		return JSON
	}
	return YAML
}

type InputFile struct {
	Format Format
	Path   string
}

type Output int

const (
	OutputInvalid Output = iota - 1
	OutputYAML
	OutputJSON
	OutputRaw // Only with --eval
)

func OutputFromString(s string) Output {
	switch strings.ToLower(s) {
	case "json", "j":
		return OutputJSON
	case "yaml", "y":
		return OutputYAML
	case "raw", "r":
		return OutputRaw
	default:
		return OutputInvalid
	}
}

type Decoder interface {
	Decode(x interface{}) error
}

func NewDecoder(r io.Reader, format Format) Decoder {
	switch format {
	case JSON:
		return json.NewDecoder(r)
	case YAML:
		return yaml.NewDecoder(r)
	default:
		panic("Invalid format")
	}
}

type Encoder interface {
	Encode(x interface{}) error
}

func NewEncoder(w io.Writer, format Output) Encoder {
	switch format {
	case OutputJSON:
		return json.NewEncoder(w)
	case OutputYAML:
		fallthrough
	default:
		return yaml.NewEncoder(w)
	}

}

func (f *InputFile) Size(_ context.Context) int {
	return 2
}

func (f *InputFile) Run(s Stream) error {
	format := f.Format
	if format == Auto {
		format = DetectFormat(f.Path)
	}
	var r io.Reader
	switch f.Path {
	case "", "-":
		r = os.Stdin
	default:
		f, err := os.Open(f.Path)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}
	return ReadFromTask(r, format).Run(s)
}
