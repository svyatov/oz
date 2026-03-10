package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFieldValueKinds(t *testing.T) {
	tests := []struct {
		name      string
		val       FieldValue
		isString  bool
		isBool    bool
		isStrings bool
	}{
		{"string", StringVal("hello"), true, false, false},
		{"bool", BoolVal(true), false, true, false},
		{"strings", StringsVal("a", "b"), false, false, true},
		{"zero_value", FieldValue{}, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.val.IsString() != tt.isString {
				t.Errorf("IsString() = %v, want %v", tt.val.IsString(), tt.isString)
			}
			if tt.val.IsBool() != tt.isBool {
				t.Errorf("IsBool() = %v, want %v", tt.val.IsBool(), tt.isBool)
			}
			if tt.val.IsStrings() != tt.isStrings {
				t.Errorf("IsStrings() = %v, want %v", tt.val.IsStrings(), tt.isStrings)
			}
		})
	}
}

func TestFieldValueAccessors(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		v := StringVal("hello")
		if v.String() != "hello" {
			t.Errorf("String() = %q", v.String())
		}
	})
	t.Run("bool", func(t *testing.T) {
		v := BoolVal(true)
		if !v.Bool() {
			t.Error("Bool() = false")
		}
	})
	t.Run("strings", func(t *testing.T) {
		v := StringsVal("a", "b")
		ss := v.Strings()
		if len(ss) != 2 || ss[0] != "a" || ss[1] != "b" {
			t.Errorf("Strings() = %v", ss)
		}
	})
}

func TestFieldValueScalar(t *testing.T) {
	tests := []struct {
		name string
		val  FieldValue
		want string
	}{
		{"string", StringVal("foo"), "foo"},
		{"bool_true", BoolVal(true), "true"},
		{"bool_false", BoolVal(false), "false"},
		{"strings_empty", StringsVal("a"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.val.Scalar(); got != tt.want {
				t.Errorf("Scalar() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFieldValueDisplay(t *testing.T) {
	tests := []struct {
		name string
		val  FieldValue
		want string
	}{
		{"string", StringVal("foo"), "foo"},
		{"bool_true", BoolVal(true), "true"},
		{"strings", StringsVal("a", "b"), "a, b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.val.Display(); got != tt.want {
				t.Errorf("Display() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFieldValueYAMLRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		val  FieldValue
	}{
		{"string", StringVal("hello")},
		{"bool_true", BoolVal(true)},
		{"bool_false", BoolVal(false)},
		{"strings", StringsVal("a", "b")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := yaml.Marshal(tt.val)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var got FieldValue
			if err := yaml.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if got.Kind() != tt.val.Kind() {
				t.Errorf("kind = %v, want %v", got.Kind(), tt.val.Kind())
			}
			if got.Display() != tt.val.Display() {
				t.Errorf("display = %q, want %q", got.Display(), tt.val.Display())
			}
		})
	}
}

func TestValuesYAMLRoundTrip(t *testing.T) {
	vals := Values{
		"lang":    StringVal("go"),
		"verbose": BoolVal(true),
		"tags":    StringsVal("api", "web"),
	}
	data, err := yaml.Marshal(vals)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got Values
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got["lang"].String() != "go" {
		t.Errorf("lang = %q", got["lang"].String())
	}
	if !got["verbose"].Bool() {
		t.Error("verbose = false")
	}
	ss := got["tags"].Strings()
	if len(ss) != 2 || ss[0] != "api" || ss[1] != "web" {
		t.Errorf("tags = %v", ss)
	}
}
