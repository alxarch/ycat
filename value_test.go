package ycat_test

import (
	"reflect"
	"testing"

	"github.com/alxarch/ycat"
	yaml "gopkg.in/yaml.v2"
)

func TestValue_UnmarshalYAML(t *testing.T) {
	type TestCase struct {
		YAML  string
		Type  ycat.ValueType
		Value interface{}
		Error bool
	}
	for _, tc := range []TestCase{
		{`null`, ycat.Null, (interface{})(nil), false},
		{``, ycat.Null, (interface{})(nil), false},
		{`foo: bar`, ycat.Object, map[string]interface{}{"foo": "bar"}, false},
		{`[1,2,3]`, ycat.Array, []interface{}{1, 2, 3}, false},
		{`42.0`, ycat.Number, 42.0, false},
		{`"foo"`, ycat.String, "foo", false},
		{`true`, ycat.Boolean, true, false},
		{`false`, ycat.Boolean, false, false},
	} {
		v := ycat.Value{}
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
