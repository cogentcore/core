// Code generated by "enumgen"; DO NOT EDIT.

package vphong

import (
	"errors"
	"strconv"
	"strings"

	"goki.dev/enums"
)

var _SetsValues = []Sets{0, 1, 2, 3}

// SetsN is the highest valid value
// for type Sets, plus one.
const SetsN Sets = 4

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the enumgen command to generate them again.
func _SetsNoOp() {
	var x [1]struct{}
	_ = x[MtxsSet-(0)]
	_ = x[NLightSet-(1)]
	_ = x[LightSet-(2)]
	_ = x[TexSet-(3)]
}

var _SetsNameToValueMap = map[string]Sets{
	`MtxsSet`:   0,
	`mtxsset`:   0,
	`NLightSet`: 1,
	`nlightset`: 1,
	`LightSet`:  2,
	`lightset`:  2,
	`TexSet`:    3,
	`texset`:    3,
}

var _SetsDescMap = map[Sets]string{
	0: ``,
	1: ``,
	2: ``,
	3: ``,
}

var _SetsMap = map[Sets]string{
	0: `MtxsSet`,
	1: `NLightSet`,
	2: `LightSet`,
	3: `TexSet`,
}

// String returns the string representation
// of this Sets value.
func (i Sets) String() string {
	if str, ok := _SetsMap[i]; ok {
		return str
	}
	return strconv.FormatInt(int64(i), 10)
}

// SetString sets the Sets value from its
// string representation, and returns an
// error if the string is invalid.
func (i *Sets) SetString(s string) error {
	if val, ok := _SetsNameToValueMap[s]; ok {
		*i = val
		return nil
	}
	if val, ok := _SetsNameToValueMap[strings.ToLower(s)]; ok {
		*i = val
		return nil
	}
	return errors.New(s + " is not a valid value for type Sets")
}

// Int64 returns the Sets value as an int64.
func (i Sets) Int64() int64 {
	return int64(i)
}

// SetInt64 sets the Sets value from an int64.
func (i *Sets) SetInt64(in int64) {
	*i = Sets(in)
}

// Desc returns the description of the Sets value.
func (i Sets) Desc() string {
	if str, ok := _SetsDescMap[i]; ok {
		return str
	}
	return i.String()
}

// SetsValues returns all possible values
// for the type Sets.
func SetsValues() []Sets {
	return _SetsValues
}

// Values returns all possible values
// for the type Sets.
func (i Sets) Values() []enums.Enum {
	res := make([]enums.Enum, len(_SetsValues))
	for i, d := range _SetsValues {
		res[i] = d
	}
	return res
}

// IsValid returns whether the value is a
// valid option for type Sets.
func (i Sets) IsValid() bool {
	_, ok := _SetsMap[i]
	return ok
}

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i Sets) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *Sets) UnmarshalText(text []byte) error {
	return i.SetString(string(text))
}
