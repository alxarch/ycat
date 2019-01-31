package yjq

import "encoding/json"

type ValueType int

const (
	Null ValueType = iota
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

type Value struct {
	Type   ValueType
	Value  interface{}
	Object map[string]interface{}
	Array  []interface{}
	String string
	Number float64
	Bool   bool
}

func TypeOf(x interface{}) ValueType {
	if x == nil {
		return Null
	}
	switch x.(type) {
	case map[string]interface{}:
		return Object
	case []interface{}:
		return Array
	case string:
		return String
	case float64:
		return Number
	case bool:
		return Boolean
	default:
		return 0
	}
}
func (v *Value) UnmarshalJSON(data []byte) error {
	var x interface{}
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	v.Type = TypeOf(x)
	v.Value = x
	return nil
}

func (v *Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Value)
}

func (v *Value) IsZero() bool {
	return v.Type == Null
}
func (v *Value) MarshalYAML() (interface{}, error) {
	return v.Value, nil
}
func (v *Value) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var obj map[string]interface{}
	if err = unmarshal(&obj); err == nil {
		v.Value = obj
		v.Type = Object
		return nil
	}
	var arr []interface{}
	if err = unmarshal(&arr); err == nil {
		v.Value = arr
		v.Type = Array
		return nil
	}
	var b bool
	if err = unmarshal(&b); err == nil {
		v.Value = b
		v.Type = Boolean
		return nil
	}
	var num float64
	if err = unmarshal(&num); err == nil {
		v.Value = num
		v.Type = Number
		return nil
	}
	var str string
	if err = unmarshal(&str); err == nil {
		v.Value = str
		v.Type = String
		return nil
	}
	return err

}
