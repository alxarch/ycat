package ycat

import (
	"encoding/json"
	"io"
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

func NewEncoder(w io.Writer, format Format) Encoder {
	switch format {
	case JSON:
		return json.NewEncoder(w)
	case YAML:
		return yaml.NewEncoder(w)
	default:
		panic("Invalid format")
	}

}
