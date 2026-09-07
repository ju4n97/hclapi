package star

import (
	"errors"
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

// CaseInsensitiveDict implements a Python-compatible dictionary with RFC 9110
// case-insensitive lookups, sequence iteration, and dot-notation attribute access.
type CaseInsensitiveDict struct {
	m map[string]starlark.Value
}

// NewCaseInsensitiveDict creates an instance from an arbitrary Go map.
func NewCaseInsensitiveDict(data map[string]any) *CaseInsensitiveDict {
	m := make(map[string]starlark.Value, len(data))
	for k, v := range data {
		m[strings.ToLower(k)] = ToValue(v)
	}
	return &CaseInsensitiveDict{m: m}
}

// NewCaseInsensitiveDictFromStrings creates an instance from string-keyed headers.
func NewCaseInsensitiveDictFromStrings(data map[string]string) *CaseInsensitiveDict {
	m := make(map[string]starlark.Value, len(data))
	for k, v := range data {
		m[strings.ToLower(k)] = starlark.String(v)
	}
	return &CaseInsensitiveDict{m: m}
}

// ToMap converts the CaseInsensitiveDict to a Go map.
func (d *CaseInsensitiveDict) ToMap() map[string]any {
	res := make(map[string]any, len(d.m))
	for k, v := range d.m {
		res[k] = ToGo(v)
	}
	return res
}

// String implements [starlark.Value.String].
func (d *CaseInsensitiveDict) String() string {
	parts := make([]string, 0, len(d.m))
	for k, v := range d.m {
		parts = append(parts, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// Type implements [starlark.Value.Type].
func (d *CaseInsensitiveDict) Type() string {
	return "case_insensitive_dict"
}

// Freeze implements [starlark.Value.Freeze].
func (d *CaseInsensitiveDict) Freeze() {}

// Truth implements [starlark.Value.Truth].
func (d *CaseInsensitiveDict) Truth() starlark.Bool {
	return len(d.m) > 0
}

// Hash implements [starlark.Value.Hash].
func (d *CaseInsensitiveDict) Hash() (uint32, error) {
	return 0, errors.New("unhashable type: case_insensitive_dict")
}

// Get implements starlark.Mapping: supports `d["Key"]` and `"Key" in d`.
func (d *CaseInsensitiveDict) Get(k starlark.Value) (starlark.Value, bool, error) {
	s, ok := k.(starlark.String)
	if !ok {
		return starlark.None, false, nil
	}
	val, found := d.m[strings.ToLower(string(s))]
	if !found {
		return starlark.None, false, nil
	}
	return val, true, nil
}

// Attr implements [starlark.HasAttrs].
// Supports `.get()`, `.keys()`, `.values()`, `.items()`, and dot-notation.
func (d *CaseInsensitiveDict) Attr(name string) (starlark.Value, error) {
	switch name {
	case "get":
		return starlark.NewBuiltin("get", func(
			thread *starlark.Thread,
			b *starlark.Builtin,
			args starlark.Tuple,
			kwargs []starlark.Tuple,
		) (starlark.Value, error) {
			var key string
			var dflt starlark.Value = starlark.None
			if err := starlark.UnpackPositionalArgs("get", args, kwargs, 1, &key, &dflt); err != nil {
				return nil, err
			}
			if val, ok := d.m[strings.ToLower(key)]; ok {
				return val, nil
			}
			return dflt, nil
		}), nil

	case "keys":
		return starlark.NewBuiltin("keys", func(
			thread *starlark.Thread,
			b *starlark.Builtin,
			args starlark.Tuple,
			kwargs []starlark.Tuple,
		) (starlark.Value, error) {
			list := make([]starlark.Value, 0, len(d.m))
			for k := range d.m {
				list = append(list, starlark.String(k))
			}
			return starlark.NewList(list), nil
		}), nil

	case "values":
		return starlark.NewBuiltin("values", func(
			thread *starlark.Thread,
			b *starlark.Builtin,
			args starlark.Tuple,
			kwargs []starlark.Tuple,
		) (starlark.Value, error) {
			list := make([]starlark.Value, 0, len(d.m))
			for _, v := range d.m {
				list = append(list, v)
			}
			return starlark.NewList(list), nil
		}), nil

	case "items":
		return starlark.NewBuiltin("items", func(
			thread *starlark.Thread,
			b *starlark.Builtin,
			args starlark.Tuple,
			kwargs []starlark.Tuple,
		) (starlark.Value, error) {
			list := make([]starlark.Value, 0, len(d.m))
			for k, v := range d.m {
				list = append(list, starlark.Tuple{starlark.String(k), v})
			}
			return starlark.NewList(list), nil
		}), nil
	}

	// Dot-notation property access: d.authorization
	if val, ok := d.m[strings.ToLower(name)]; ok {
		return val, nil
	}

	return nil, nil
}

// AttrNames implements [starlark.Value.AttrNames].
func (d *CaseInsensitiveDict) AttrNames() []string {
	names := []string{"get", "keys", "values", "items"}
	for k := range d.m {
		names = append(names, k)
	}
	return names
}

// Iterate implements [starlark.Value.Iterate].
func (d *CaseInsensitiveDict) Iterate() starlark.Iterator {
	keys := make([]starlark.Value, 0, len(d.m))
	for k := range d.m {
		keys = append(keys, starlark.String(k))
	}
	return starlark.NewList(keys).Iterate()
}

// Len implements [starlark.Value.Len].
func (d *CaseInsensitiveDict) Len() int {
	return len(d.m)
}
