package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// valueKind discriminates the three possible field value types.
type valueKind uint8

const (
	StringKind  valueKind = iota // zero value = empty string
	BoolKind
	StringsKind
)

// FieldValue is a type-safe sum type for wizard field values.
// A field value is always one of: string, bool, or []string.
//
//nolint:recvcheck // UnmarshalYAML requires pointer receiver; read-only methods use value receiver by design
type FieldValue struct {
	s    string
	ss   []string
	b    bool
	kind valueKind
}

// StringVal creates a string FieldValue.
func StringVal(s string) FieldValue {
	return FieldValue{s: s, kind: StringKind}
}

// BoolVal creates a bool FieldValue.
func BoolVal(b bool) FieldValue {
	return FieldValue{b: b, kind: BoolKind}
}

// StringsVal creates a []string FieldValue.
func StringsVal(ss ...string) FieldValue {
	return FieldValue{ss: ss, kind: StringsKind}
}

// Kind returns the discriminant.
func (v FieldValue) Kind() valueKind { return v.kind }

// IsString returns true if the value is a string.
func (v FieldValue) IsString() bool { return v.kind == StringKind }

// IsBool returns true if the value is a bool.
func (v FieldValue) IsBool() bool { return v.kind == BoolKind }

// IsStrings returns true if the value is a []string.
func (v FieldValue) IsStrings() bool { return v.kind == StringsKind }

// String returns the string value. Only meaningful for StringKind.
func (v FieldValue) String() string { return v.s }

// Bool returns the bool value. Only meaningful for BoolKind.
func (v FieldValue) Bool() bool { return v.b }

// Strings returns the []string value. Only meaningful for StringsKind.
func (v FieldValue) Strings() []string { return v.ss }

// Scalar returns a string representation of any scalar value.
// Bool → "true"/"false", String → itself. For StringsKind, returns "".
func (v FieldValue) Scalar() string {
	switch v.kind {
	case BoolKind:
		if v.b {
			return "true"
		}
		return "false"
	case StringKind:
		return v.s
	case StringsKind:
		return ""
	}
	return ""
}

// Display returns a human-readable representation.
// Strings join with ", ".
func (v FieldValue) Display() string {
	switch v.kind {
	case BoolKind:
		if v.b {
			return "true"
		}
		return "false"
	case StringsKind:
		return strings.Join(v.ss, ", ")
	case StringKind:
		return v.s
	}
	return v.s
}

// MarshalYAML returns the native Go type for YAML serialization.
func (v FieldValue) MarshalYAML() (any, error) {
	switch v.kind {
	case BoolKind:
		return v.b, nil
	case StringsKind:
		return v.ss, nil
	case StringKind:
		return v.s, nil
	}
	return v.s, nil
}

// UnmarshalYAML decodes a YAML node into a FieldValue.
func (v *FieldValue) UnmarshalYAML(node *yaml.Node) error {
	//nolint:exhaustive // only ScalarNode and SequenceNode are valid for FieldValue
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag == "!!bool" {
			var b bool
			if err := node.Decode(&b); err != nil {
				return fmt.Errorf("decoding bool: %w", err)
			}
			*v = BoolVal(b)
			return nil
		}
		*v = StringVal(node.Value)
		return nil
	case yaml.SequenceNode:
		var ss []string
		if err := node.Decode(&ss); err != nil {
			return fmt.Errorf("decoding string slice: %w", err)
		}
		*v = StringsVal(ss...)
		return nil
	default:
		return fmt.Errorf("unsupported YAML node kind %v for FieldValue", node.Kind)
	}
}

// Values maps option names to field values.
type Values map[string]FieldValue
