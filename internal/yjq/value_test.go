package yjq_test

import (
	"reflect"
	"testing"

	"github.com/alxarch/yjq/internal/yjq"
	yaml "gopkg.in/yaml.v2"
)

func TestValue_UnmarshalYAML(t *testing.T) {
	type TestCase struct {
		YAML  string
		Type  yjq.ValueType
		Value interface{}
		Error bool
	}
	for _, tc := range []TestCase{
		{`null`, yjq.Null, (interface{})(nil), false},
		{``, yjq.Null, (interface{})(nil), false},
		{`foo: bar`, yjq.Object, map[string]interface{}{"foo": "bar"}, false},
		{`[1,2,3]`, yjq.Array, []interface{}{1, 2, 3}, false},
		{`42.0`, yjq.Number, 42.0, false},
		{`"foo"`, yjq.String, "foo", false},
		{`true`, yjq.Boolean, true, false},
		{`false`, yjq.Boolean, false, false},
	} {
		v := yjq.Value{}
		err := yaml.Unmarshal([]byte(tc.YAML), &v)
		if err != nil {
			t.Errorf("%q Unexpected error: %s", tc.YAML, err)
		}
		if v.Type != tc.Type {
			t.Errorf("%q Invalid type : %s != %s", tc.YAML, v.Type, tc.Type)
		}
		if !reflect.DeepEqual(v.Value, tc.Value) {
			t.Errorf("%q Invalid value: %v", tc.YAML, v.Value)
		}

	}

}
