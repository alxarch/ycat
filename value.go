package ycat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// RawValue is a JSON string
type RawValue string

// Kind returns the value's type
func (v RawValue) Kind() ValueType {
	if len(v) > 0 {
		return kindOfJSON(v[0])
	}
	return Invalid
}

// UnmarshalJSON implements json.Unmarshaler for RawValue
func (v *RawValue) UnmarshalJSON(data []byte) error {
	*v = RawValue(data)
	return nil
}

var nullBytes = []byte(`null`)

// MarshalJSON implements json.Marshaler for RawValue
func (v RawValue) MarshalJSON() ([]byte, error) {
	if v == "" {
		return nullBytes, nil
	}
	return []byte(v), nil
}
func (v RawValue) String() string {
	return string(v)
}

// Compact removes insignificant white space
func (v RawValue) Compact() (RawValue, error) {
	if v == "" {
		return v, nil
	}
	buf := bytes.NewBufferString(v.String())
	buf.Reset()
	if err := json.Compact(buf, []byte(v)); err != nil {
		return "", err
	}
	return RawValue(buf.String()), nil
}

// RawValueArray joins RawValues to an array
func RawValueArray(values ...RawValue) RawValue {
	w := strings.Builder{}
	size := len(values) + 2
	for _, v := range values {
		size += len(v)
	}
	w.Grow(size)
	w.WriteByte('[')
	for i, v := range values {
		if i > 0 {
			w.WriteByte(',')
		}
		w.WriteString(string(v))
	}
	w.WriteByte(']')
	return RawValue(w.String())
}

// MarshalJSONString marshals any value to JSON string
func MarshalJSONString(x interface{}) (string, error) {
	w := strings.Builder{}
	enc := json.NewEncoder(&w)
	err := enc.Encode(x)
	return w.String(), err
}

// NewRawValue creates a new RawValue from any value
func NewRawValue(x interface{}) (RawValue, error) {
	if x, ok := x.(JSONStringMarshaler); ok {
		v, err := x.MarshalJSONString()
		return RawValue(v), err
	}
	v, err := MarshalJSONString(x)
	return RawValue(v), err
}

// MarshalYAML implements yaml.Marshaler for RawValue
func (v RawValue) MarshalYAML() (x interface{}, err error) {
	if v == "" {
		return
	}
	r := strings.NewReader(string(v))
	dec := json.NewDecoder(r)
	dec.UseNumber()
	x, err = decodeValue(dec)
	if err != nil {
		return
	}
	// YAML pkg doesn't like json.Number on top level
	if n, ok := x.(json.Number); ok {
		f, _ := n.Float64()
		return f, nil
	}
	return

}

// UnmarshalYAML implements yaml.Unmarshaler for RawValue
func (v *RawValue) UnmarshalYAML(fn func(interface{}) error) (err error) {
	var a []RawValue
	if err = fn(&a); err == nil {
		if a == nil {
			*v = "[]"
		} else {
			*v = RawValueArray(a...)
		}
		return
	}
	var m yaml.MapSlice
	if err = fn(&m); err == nil {
		if m == nil {
			*v = "{}"
		} else {
			*v, err = NewRawValue(Map(m))
		}
		return
	}
	var b bool
	if err = fn(&b); err == nil {
		if b {
			*v = "true"
		} else {
			*v = "false"
		}
		return nil
	}
	var f float64
	if err = fn(&f); err == nil {
		var s string
		if err = fn(&s); err == nil {
			*v = RawValue(s)
			return
		}
		*v, err = NewRawValue(f)
		return
	}
	var s string
	if err = fn(&s); err == nil {
		*v, err = NewRawValue(s)
		return
	}
	return
}

// Map wraps yaml.MapSlice to provide json Marshaler/Unmarshaller
type Map yaml.MapSlice

// NewMap  creates a Map from key/value pairs
func NewMap(pairs ...interface{}) (m Map) {
	var k, v interface{}
	for len(pairs) >= 2 {
		k, v, pairs = pairs[0], pairs[1], pairs[2:]
		m = append(m, yaml.MapItem{
			Key:   k,
			Value: v,
		})
	}
	if m == nil {
		return emptyMap()
	}
	return
}

// MarshalYAML implements yaml.Marshaler for Map
func (m Map) MarshalYAML() (interface{}, error) {
	return yaml.MapSlice(m), nil
}

// UnmarshalYAML implements yaml.Unmarshaler for Map
func (m *Map) UnmarshalYAML(fn func(x interface{}) error) error {
	return fn((*yaml.MapSlice)(m))
}

func decodeArray(dec *json.Decoder, arr []interface{}) ([]interface{}, error) {
	for dec.More() {
		v, err := decodeValue(dec)
		if err != nil {
			return arr, err
		}
		arr = append(arr, v)
	}
	// Consume closing token
	_, err := dec.Token()
	return arr, err
}

func decodeMap(dec *json.Decoder, m Map) (Map, error) {
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return m, err
		}
		key, ok := token.(string)
		if !ok {
			return m, fmt.Errorf("Invalid JSON key token %v", token)
		}
		value, err := decodeValue(dec)
		if err != nil {
			return m, err
		}
		m = append(m, yaml.MapItem{Key: key, Value: value})
	}
	// Consume closing token
	_, err := dec.Token()
	return m, err

}
func emptyMap() Map {
	return make([]yaml.MapItem, 0)
}
func decodeValue(dec *json.Decoder) (v interface{}, err error) {
	token, err := dec.Token()
	if err != nil {
		return
	}
	if token == nil {
		// Null value
		return
	}
	switch token := token.(type) {
	case json.Delim:
		switch token {
		case '[':
			arr, err := decodeArray(dec, nil)
			if err != nil {
				return nil, err
			}
			if arr == nil {
				arr = make([]interface{}, 0)
			}
			v = arr
		case '{':
			m, err := decodeMap(dec, nil)
			if err != nil {
				return nil, err
			}
			if m == nil {
				m = emptyMap()
			}
			v = m
		default:
			return nil, fmt.Errorf("Invalid token %q", token)
		}
	default:
		v = token
	}
	return
}

// UnmarshalJSON implements json.Unmarshaler for Map
func (m *Map) UnmarshalJSON(data []byte) (err error) {
	r := bytes.NewReader(data)
	dec := json.NewDecoder(r)
	token, err := dec.Token()
	if err != nil {
		return
	}
	if token == nil {
		*m = nil
		return
	}
	if d, ok := token.(json.Delim); ok && d == '{' {
		*m, err = decodeMap(dec, (*m)[:0])
		if *m == nil {
			*m = emptyMap()
		}
		return
	}
	err = fmt.Errorf("Invalid token %q", token)
	return
}

type writer interface {
	io.ByteWriter
	io.Writer
}
type writerJSON interface {
	writeJSON(enc *json.Encoder, w writer) error
}

func (m Map) writeJSON(enc *json.Encoder, w writer) (err error) {
	if m == nil {
		return enc.Encode(nil)
	}
	if err = w.WriteByte('{'); err != nil {
		return
	}
	for i := range m {
		if i > 0 {
			if err = w.WriteByte(','); err != nil {
				return
			}
		}
		item := &m[i]
		key, ok := item.Key.(string)
		if !ok {
			err = fmt.Errorf("Invalid key %v", item.Key)
			return
		}
		if err = enc.Encode(key); err != nil {
			return
		}
		if err = w.WriteByte(':'); err != nil {
			return
		}
		if err = writeJSON(enc, w, item.Value); err != nil {
			return
		}
	}
	err = w.WriteByte('}')
	return
}
func writeJSON(enc *json.Encoder, w writer, x interface{}) (err error) {
	switch v := x.(type) {
	case yaml.MapSlice:
		if v == nil {
			_, err = io.WriteString(w, "{}")
			return
		}
		return Map(v).writeJSON(enc, w)
	case Map:
		return v.writeJSON(enc, w)
	case []interface{}:
		if v == nil {
			return enc.Encode(nil)
		}
		if err = w.WriteByte('['); err != nil {
			return
		}
		for i, v := range v {
			if i > 0 {
				if err = w.WriteByte(','); err != nil {
					return
				}
			}

			if err = writeJSON(enc, w, v); err != nil {
				return
			}
		}
		err = w.WriteByte(']')
		return
	default:
		if jw, ok := v.(writerJSON); ok {
			return jw.writeJSON(enc, w)
		}
		return enc.Encode(v)
	}
}

//JSONStringMarshaler marshals to JSON string
type JSONStringMarshaler interface {
	MarshalJSONString() (string, error)
}

// MarshalJSONString implements JSONStringMarshaler for Map
func (m Map) MarshalJSONString() (string, error) {
	if m == nil {
		return "null", nil
	}
	w := strings.Builder{}
	enc := json.NewEncoder(&w)
	if err := m.writeJSON(enc, &w); err != nil {
		return "", err
	}
	return w.String(), nil
}

// MarshalJSON implements json.Marshaler for Map
func (m Map) MarshalJSON() ([]byte, error) {
	if m == nil {
		return nullBytes, nil
	}
	buf := bytes.NewBuffer(make([]byte, 0, 64))
	enc := json.NewEncoder(buf)
	if err := m.writeJSON(enc, buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ValueType is the JSON type of a value
type ValueType int

// Value types
const (
	Invalid ValueType = iota - 1
	Null
	Object
	Array
	String
	Number
	Boolean
)

func (t ValueType) String() string {
	switch t {
	case Object:
		return "Object"
	case Array:
		return "Array"
	case String:
		return "String"
	case Number:
		return "Number"
	case Boolean:
		return "Boolean"
	case Null:
		return "Null"
	default:
		return "Invalid"
	}
}
func kindOfJSON(c byte) ValueType {
	switch c {
	case '{':
		return Object
	case '[':
		return Array
	case '"':
		return String
	case 't', 'f':
		return Boolean
	case 'n':
		return Null
	case '.':
		return Number
	default:
		if c -= '0'; 0 <= c && c <= 9 {
			return Number
		}
		return Invalid
	}
}
