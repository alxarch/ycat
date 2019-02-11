package ycat_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/alxarch/ycat"
	yaml "gopkg.in/yaml.v2"
)

func TestMap_MarshalJSON(t *testing.T) {
	m := ycat.NewMap
	a := func(values ...interface{}) []interface{} {
		return values
	}
	tests := []struct {
		Map      ycat.Map
		wantJSON string
		wantErr  bool
	}{
		{nil, `null`, false},
		{m(), `{}`, false},
		{m("foo", m()), `{"foo":{}}`, false},
		{m("foo", a(42)), `{"foo":[42]}`, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.wantJSON), func(t *testing.T) {
			data, err := json.Marshal(tt.Map)
			if (err != nil) != tt.wantErr {
				t.Errorf("Map.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if string(data) != tt.wantJSON {
				t.Errorf("Map.MarshalJSON() %q != %q", data, tt.wantJSON)
			}
		})
	}

}
func TestMap_UnmarshalJSON(t *testing.T) {
	m := ycat.NewMap
	a := func(values ...interface{}) []interface{} {
		return values
	}
	tests := []struct {
		Map     ycat.Map
		JSON    string
		wantErr bool
	}{
		{m("foo", m("bar", "baz")), `{"foo":{"bar":"baz"}}`, false},
		{m("foo", a(1, 2, 3)), `{"foo":[1,2,3]}`, false},
		{nil, `null`, false},
		{m(), `{}`, false},
		{m("foo", m()), `{"foo":{}}`, false},
	}
	for _, tt := range tests {
		t.Run("json="+string(tt.JSON), func(t *testing.T) {
			var v ycat.Map
			err := json.Unmarshal([]byte(tt.JSON), &v)
			if (err != nil) != tt.wantErr {
				t.Errorf("map.UnmarshalJSON() error = %v, wanterr %v", err, tt.wantErr)
			}
			data, err := json.Marshal(v)
			if (err != nil) != tt.wantErr {
				t.Errorf("map.UnmarshalJSON() error = %v, wanterr %v", err, tt.wantErr)
			}

			if string(data) != tt.JSON {
				t.Errorf("Map.UnmarshalJSON() %q != %q", string(data), tt.JSON)
			}
		})
	}

}
func TestRawValue_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		YAML      string
		wantValue ycat.RawValue
		wantErr   bool
	}{
		{`foo: {}`, `{"foo":{}}`, false},
		{"{}", `{}`, false},
		{`""`, `""`, false},
		{"[]", `[]`, false},
		{"42", "42", false},
		{"foo: bar", `{"foo":"bar"}`, false},
		{"null", ``, false},
		{"[1,2,3]", `[1,2,3]`, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.wantValue), func(t *testing.T) {
			var v ycat.RawValue
			if err := yaml.Unmarshal([]byte(tt.YAML), &v); (err != nil) != tt.wantErr {
				t.Errorf("RawValue.UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
			v, err := v.Compact()
			if (err != nil) != tt.wantErr {
				t.Errorf("RawValue.Compact() error = %v, wantErr %v", err, tt.wantErr)
			}
			if v != tt.wantValue {
				t.Errorf("RawValue.UnmarshalYAML() %q != %q", v, tt.wantValue)
			}
		})
	}
}

func TestRawValue_MarshalYAML(t *testing.T) {
	tests := []struct {
		YAML    string
		value   ycat.RawValue
		wantErr bool
	}{
		{`foo: {}`, `{"foo":{}}`, false},
		{"{}", `{}`, false},
		{`""`, `""`, false},
		{"[]", "[]", false},
		{"answer: 42", `{"answer":42}`, false},
		{"answer:\n- 42\n", `{"answer":[42]}`, false},
		{"42", "42", false},
		{"foo: bar", `{"foo":"bar"}`, false},
		{"null", ``, false},
		{"- 1\n- 2\n- 3", `[1,2,3]`, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.value), func(t *testing.T) {
			data, err := yaml.Marshal(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("RawValue.MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
			want := tt.YAML
			if !strings.HasSuffix(want, "\n") {
				want += "\n"
			}

			if string(data) != want {
				t.Errorf("RawValue.MarsalYAML() %q != %q", string(data), want)
			}
		})
	}
}
