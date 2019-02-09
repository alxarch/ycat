package ycat

import (
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
	Decode(interface{}) error
}

func NewDecoder(r io.Reader, format Format) Decoder {
	switch format {
	case JSON:
		return json.NewDecoder(r)
	default:
		return yaml.NewDecoder(r)
	}
}

func ReadFromFile(path string, format Format) ProducerFunc {
	if format == Auto {
		format = DetectFormat(path)
	}
	return func(s WriteStream) error {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		r := ReadFromTask(f, format)
		return r(s)
	}
}

func ReadFromTask(r io.Reader, format Format) ProducerFunc {
	return func(s WriteStream) error {
		dec := NewDecoder(r, format)
		for {
			v := new(Value)
			if err := dec.Decode(v); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

			if !s.Push(v) {
				return nil
			}
		}
	}
}

func StreamWriteJSON(w io.WriteCloser) ConsumerFunc {
	enc := json.NewEncoder(w)
	return func(s ReadStream) error {
		defer w.Close()
		for {
			v, ok := s.Next()
			if !ok {
				return nil
			}
			if err := enc.Encode(v); err != nil {
				return err
			}
		}
	}
}

func StreamWriteYAML(w io.WriteCloser) ConsumerFunc {
	n := int64(0)
	return func(s ReadStream) error {
		defer w.Close()
		for {
			v, ok := s.Next()
			if !ok {
				return nil
			}
			if n > 0 {
				nn, err := w.Write([]byte("---\n"))
				if err != nil {
					return err
				}
				n += int64(nn)
			}
			data, err := yaml.Marshal(v)
			if err != nil {
				return err
			}
			nn, err := w.Write(data)
			if err != nil {
				return err
			}
			n += int64(nn)
		}
	}
}
