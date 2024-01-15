// Code generated by "goki generate"; DO NOT EDIT.

package images

import (
	"errors"
	"log"
	"strconv"

	"goki.dev/enums"
)

var _FormatsValues = []Formats{0, 1, 2, 3, 4, 5, 6}

// FormatsN is the highest valid value
// for type Formats, plus one.
const FormatsN Formats = 7

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the enumgen command to generate them again.
func _FormatsNoOp() {
	var x [1]struct{}
	_ = x[None-(0)]
	_ = x[PNG-(1)]
	_ = x[JPEG-(2)]
	_ = x[GIF-(3)]
	_ = x[TIFF-(4)]
	_ = x[BMP-(5)]
	_ = x[WebP-(6)]
}

var _FormatsNameToValueMap = map[string]Formats{
	`None`: 0,
	`PNG`:  1,
	`JPEG`: 2,
	`GIF`:  3,
	`TIFF`: 4,
	`BMP`:  5,
	`WebP`: 6,
}

var _FormatsDescMap = map[Formats]string{
	0: ``,
	1: ``,
	2: ``,
	3: ``,
	4: ``,
	5: ``,
	6: ``,
}

var _FormatsMap = map[Formats]string{
	0: `None`,
	1: `PNG`,
	2: `JPEG`,
	3: `GIF`,
	4: `TIFF`,
	5: `BMP`,
	6: `WebP`,
}

// String returns the string representation
// of this Formats value.
func (i Formats) String() string {
	if str, ok := _FormatsMap[i]; ok {
		return str
	}
	return strconv.FormatInt(int64(i), 10)
}

// SetString sets the Formats value from its
// string representation, and returns an
// error if the string is invalid.
func (i *Formats) SetString(s string) error {
	if val, ok := _FormatsNameToValueMap[s]; ok {
		*i = val
		return nil
	}
	return errors.New(s + " is not a valid value for type Formats")
}

// Int64 returns the Formats value as an int64.
func (i Formats) Int64() int64 {
	return int64(i)
}

// SetInt64 sets the Formats value from an int64.
func (i *Formats) SetInt64(in int64) {
	*i = Formats(in)
}

// Desc returns the description of the Formats value.
func (i Formats) Desc() string {
	if str, ok := _FormatsDescMap[i]; ok {
		return str
	}
	return i.String()
}

// FormatsValues returns all possible values
// for the type Formats.
func FormatsValues() []Formats {
	return _FormatsValues
}

// Values returns all possible values
// for the type Formats.
func (i Formats) Values() []enums.Enum {
	res := make([]enums.Enum, len(_FormatsValues))
	for i, d := range _FormatsValues {
		res[i] = d
	}
	return res
}

// IsValid returns whether the value is a
// valid option for type Formats.
func (i Formats) IsValid() bool {
	_, ok := _FormatsMap[i]
	return ok
}

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i Formats) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *Formats) UnmarshalText(text []byte) error {
	if err := i.SetString(string(text)); err != nil {
		log.Println("Formats.UnmarshalText:", err)
	}
	return nil
}
