package ycat

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// Format is input file format
type Format uint

// Input formats
const (
	Auto Format = iota
	YAML
	JSON
	JSONNET
)

// FormatFromString converts a string to Format
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

// DetectFormat detects an input format from the extension
func DetectFormat(filename string) Format {
	switch path.Ext(filename) {
	case ".json":
		return JSON
	case ".jsonnet", ".libsonnet":
		return JSONNET
	default:
		return YAML
	}
}

// Output is an output format
type Output int

// Output formats
const (
	OutputInvalid Output = iota - 1
	OutputYAML
	OutputJSON
	OutputRaw // Only with --eval
)

// OutputFromString converts a string to Output
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

// Decoder is a value decoder
type Decoder interface {
	Decode(interface{}) error
}

// NewDecoder creates a new Decoder decoding values from a Reader
func NewDecoder(r io.Reader, format Format) Decoder {
	switch format {
	case JSON:
		return json.NewDecoder(r)
	default:
		return yaml.NewDecoder(r)
	}
}

// ReadFromFile creates a StreamTask to read values from a file
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

// ReadFromTask creates a StreamTask to read values from a Reader
func ReadFromTask(r io.Reader, format Format) ProducerFunc {
	return func(s WriteStream) error {
		dec := NewDecoder(r, format)
		for {
			var v RawValue
			if err := dec.Decode(&v); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			if v == "" {
				v = "null"
			}

			if !s.Push(v) {
				return nil
			}
		}
	}
}

// StreamWriteJSON creates a StreamTask to write values as JSON to a Writer
func StreamWriteJSON(w io.WriteCloser) ConsumerFunc {
	return func(s ReadStream) error {
		defer w.Close()
		var (
			buf  = bytes.Buffer{}
			data []byte
		)
		for {
			v, ok := s.Next()
			if !ok {
				// No more stream values
				return nil
			}
			data = append(data[:0], string(v)...) // Avoid allocations

			// Compact JSON output
			buf.Reset()
			if err := json.Compact(&buf, data); err != nil {
				return err
			}
			// One value per line
			buf.WriteByte('\n')
			if _, err := buf.WriteTo(w); err != nil {
				return err
			}
		}
	}
}

// StreamWriteYAML creates a StreamTask to write values as YAML to a Writer
func StreamWriteYAML(w io.WriteCloser) ConsumerFunc {
	const newDocSeparator = "---\n"

	return func(s ReadStream) (err error) {
		// Close output when done
		// Not sure this is the responsibility of the task
		defer w.Close()
		for numValues := 0; ; numValues++ {
			v, ok := s.Next()
			if !ok {
				// No more stream values
				return
			}

			// Separate YAML documents
			if numValues > 0 {
				_, err = io.WriteString(w, newDocSeparator)
				if err != nil {
					return
				}
			}
			// Encode a single YAML document
			// Cannot use one encoder for all values because of a yaml.Encoder bug
			// with multiple documents and scalar values
			enc := yaml.NewEncoder(w)
			if err = enc.Encode(v); err != nil {
				return
			}
		}
	}
}
