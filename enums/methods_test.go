// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enums

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// it is much easier to test with an independent enum mock
type enum int64

var hasFlag = map[enum]bool{5: true, 3: true}
var bitIndexString = map[enum]string{5: "Apple", 3: "bitIndexString", 1: "one"}

func (e enum) String() string                 { return "extended" }
func (e enum) Int64() int64                   { return int64(e) }
func (e enum) Desc() string                   { return "extendedDesc" }
func (e enum) Values() []Enum                 { return nil }
func (e enum) HasFlag(f BitFlag) bool         { return hasFlag[f.(enum)] }
func (e enum) BitIndexString() string         { return bitIndexString[e] }
func (e *enum) SetInt64(i int64)              { *e = enum(i) }
func (e *enum) SetFlag(on bool, f ...BitFlag) { SetFlag((*int64)(e), on, f...) }
func (e *enum) SetString(s string) error {
	if s == "Orange" {
		*e = 7
		return nil
	}
	return errors.New("invalid")
}
func (e *enum) SetStringOr(s string) error {
	if s == "Pear" {
		*e = 3
		return nil
	}
	return errors.New("invalid")
}

func TestString(t *testing.T) {
	m := map[enum]string{5: "Apple"}

	assert.Equal(t, "Apple", String(5, m))
	assert.Equal(t, "3", String(3, m))

	assert.Equal(t, "Apple", StringExtended[enum, enum](5, m))
	assert.Equal(t, "extended", StringExtended[enum, enum](3, m))

	assert.Equal(t, "Apple", BitIndexStringExtended[enum, enum](5, m))
	assert.Equal(t, "bitIndexString", BitIndexStringExtended[enum, enum](3, m))

	assert.Equal(t, "", BitFlagString(0, []enum{}))
	assert.Equal(t, "", BitFlagString(0, []enum{1}))
	assert.Equal(t, "", BitFlagString(0, []enum{1, 2, 47}))
	assert.Equal(t, "bitIndexString", BitFlagString(0, []enum{3}))
	assert.Equal(t, "Apple", BitFlagString(0, []enum{5}))
	assert.Equal(t, "bitIndexString|Apple", BitFlagString(0, []enum{3, 5}))
	assert.Equal(t, "Apple|bitIndexString", BitFlagString(0, []enum{5, 3}))
	assert.Equal(t, "Apple|bitIndexString", BitFlagString(0, []enum{5, 1, 3}))

	assert.Equal(t, "", BitFlagStringExtended(0, []enum{}, []enum{}))
	assert.Equal(t, "", BitFlagStringExtended(0, []enum{1}, []enum{2, 1}))
	assert.Equal(t, "Apple", BitFlagStringExtended(0, []enum{5}, []enum{1}))
	assert.Equal(t, "bitIndexString", BitFlagStringExtended(0, []enum{}, []enum{3}))
	assert.Equal(t, "Apple|bitIndexString", BitFlagStringExtended(0, []enum{3}, []enum{5}))
	assert.Equal(t, "bitIndexString|Apple|bitIndexString", BitFlagStringExtended(0, []enum{3}, []enum{3, 5}))
	assert.Equal(t, "bitIndexString|Apple", BitFlagStringExtended(0, []enum{5, 1}, []enum{2, 3, 1}))
}

func TestSetString(t *testing.T) {
	valueMap := map[string]enum{"apple": 5}

	i := enum(0)
	assert.NoError(t, SetString(&i, "apple", valueMap, "Fruits"))
	assert.Equal(t, enum(5), i)
	i = enum(4)
	err := SetString(&i, "Apple", valueMap, "Fruits")
	if assert.Error(t, err) {
		assert.Equal(t, "Apple is not a valid value for type Fruits", err.Error())
	}
	assert.Equal(t, enum(4), i)
	err = SetString(&i, "Orange", valueMap, "Fruits")
	if assert.Error(t, err) {
		assert.Equal(t, "Orange is not a valid value for type Fruits", err.Error())
	}
	assert.Equal(t, enum(4), i)

	assert.NoError(t, SetStringLower(&i, "apple", valueMap, "Fruits"))
	assert.Equal(t, enum(5), i)
	i = enum(4)
	assert.NoError(t, SetStringLower(&i, "Apple", valueMap, "Fruits"))
	assert.Equal(t, enum(5), i)
	i = enum(4)
	err = SetStringLower(&i, "Orange", valueMap, "Fruits")
	if assert.Error(t, err) {
		assert.Equal(t, "Orange is not a valid value for type Fruits", err.Error())
	}
	assert.Equal(t, enum(4), i)

	assert.NoError(t, SetStringExtended(&i, &i, "apple", valueMap))
	assert.Equal(t, enum(5), i)
	i = enum(4)
	assert.NoError(t, SetStringExtended(&i, &i, "Orange", valueMap))
	assert.Equal(t, enum(7), i)
	i = enum(4)
	err = SetStringExtended(&i, &i, "Apple", valueMap)
	if assert.Error(t, err) {
		assert.Equal(t, "invalid", err.Error())
	}
	assert.Equal(t, enum(4), i)

	assert.NoError(t, SetStringLowerExtended(&i, &i, "apple", valueMap))
	assert.Equal(t, enum(5), i)
	i = enum(4)
	assert.NoError(t, SetStringLowerExtended(&i, &i, "Apple", valueMap))
	assert.Equal(t, enum(5), i)
	i = enum(4)
	assert.NoError(t, SetStringLowerExtended(&i, &i, "Orange", valueMap))
	assert.Equal(t, enum(7), i)
	i = enum(4)
	err = SetStringLowerExtended(&i, &i, "Strawberry", valueMap)
	if assert.Error(t, err) {
		assert.Equal(t, "invalid", err.Error())
	}
	assert.Equal(t, enum(4), i)
}

func TestSetStringOr(t *testing.T) {
	valueMap := map[string]enum{"apple": 5, "Orange": 3}

	i := enum(0)
	assert.NoError(t, SetStringOr(&i, "apple", valueMap, "Fruits"))
	assert.Equal(t, enum(32), i)
	assert.NoError(t, SetStringOr(&i, "Orange", valueMap, "Fruits"))
	assert.Equal(t, enum(40), i)
	i = enum(0)
	assert.NoError(t, SetStringOr(&i, "apple|Orange", valueMap, "Fruits"))
	assert.Equal(t, enum(40), i)
	assert.Error(t, SetStringOr(&i, "Apple", valueMap, "Fruits"))
	assert.Error(t, SetStringOr(&i, "Apple|Orange", valueMap, "Fruits"))
	assert.Error(t, SetStringOr(&i, "apple|Orange|Pear", valueMap, "Fruits"))

	i = enum(0)
	assert.NoError(t, SetStringOrLower(&i, "apple", valueMap, "Fruits"))
	assert.Equal(t, enum(32), i)
	assert.NoError(t, SetStringOrLower(&i, "Orange", valueMap, "Fruits"))
	assert.Equal(t, enum(40), i)
	i = enum(0)
	assert.NoError(t, SetStringOrLower(&i, "apple|Orange", valueMap, "Fruits"))
	assert.Equal(t, enum(40), i)
	i = enum(0)
	assert.NoError(t, SetStringOrLower(&i, "Apple", valueMap, "Fruits"))
	assert.Equal(t, enum(32), i)
	i = enum(0)
	assert.NoError(t, SetStringOrLower(&i, "Apple|Orange", valueMap, "Fruits"))
	assert.Equal(t, enum(40), i)
	assert.Error(t, SetStringOrLower(&i, "strawberry", valueMap, "Fruits"))
	assert.Error(t, SetStringOrLower(&i, "apple|Orange|Pear", valueMap, "Fruits"))

	i = enum(0)
	assert.NoError(t, SetStringOrExtended(&i, &i, "apple", valueMap))
	assert.Equal(t, enum(32), i)
	assert.NoError(t, SetStringOrExtended(&i, &i, "Orange", valueMap))
	assert.Equal(t, enum(40), i)
	i = enum(0)
	assert.NoError(t, SetStringOrExtended(&i, &i, "apple|Orange", valueMap))
	assert.Equal(t, enum(40), i)
	assert.NoError(t, SetStringOrExtended(&i, &i, "Pear", valueMap))
	assert.Equal(t, enum(3), i)
	assert.NoError(t, SetStringOrExtended(&i, &i, "Orange|Pear", valueMap))
	assert.Equal(t, enum(3), i)
	assert.Error(t, SetStringOrExtended(&i, &i, "Apple", valueMap))
	assert.Error(t, SetStringOrExtended(&i, &i, "Apple|Orange", valueMap))
	assert.Error(t, SetStringOrExtended(&i, &i, "apple|Orange|Peach", valueMap))

	i = enum(0)
	assert.NoError(t, SetStringOrLowerExtended(&i, &i, "apple", valueMap))
	assert.Equal(t, enum(32), i)
	assert.NoError(t, SetStringOrLowerExtended(&i, &i, "Orange", valueMap))
	assert.Equal(t, enum(40), i)
	i = enum(0)
	assert.NoError(t, SetStringOrLowerExtended(&i, &i, "apple|Orange", valueMap))
	assert.Equal(t, enum(40), i)
	assert.NoError(t, SetStringOrLowerExtended(&i, &i, "Pear", valueMap))
	assert.Equal(t, enum(3), i)
	assert.NoError(t, SetStringOrLowerExtended(&i, &i, "Orange|Pear", valueMap))
	assert.Equal(t, enum(3), i)
	i = enum(0)
	assert.NoError(t, SetStringOrLowerExtended(&i, &i, "Apple", valueMap))
	assert.Equal(t, enum(32), i)
	assert.NoError(t, SetStringOrLowerExtended(&i, &i, "Apple|Orange", valueMap))
	assert.Equal(t, enum(40), i)
	assert.Error(t, SetStringOrLowerExtended(&i, &i, "apple|Orange|Peach", valueMap))
}

func TestDesc(t *testing.T) {
	descMap := map[enum]string{5: "A red fruit"}

	assert.Equal(t, "A red fruit", Desc(enum(5), descMap))
	assert.Equal(t, "extended", Desc(enum(3), descMap))

	assert.Equal(t, "A red fruit", DescExtended[enum, enum](enum(5), descMap))
	assert.Equal(t, "extendedDesc", DescExtended[enum, enum](enum(3), descMap))
}

func TestValues(t *testing.T) {
	assert.Equal(t, []enum{1, 5, 2, 3}, ValuesGlobalExtended([]enum{2, 3}, []enum{1, 5}))
	assert.Equal(t, []Enum{enum(7), enum(4)}, Values([]enum{7, 4}))
	assert.Equal(t, []Enum{enum(7), enum(4), enum(8), enum(1)}, ValuesExtended([]enum{8, 1}, []enum{7, 4}))
}

func TestHasFlag(t *testing.T) {
	i := enum(20)
	pi := (*int64)(&i)

	assert.True(t, HasFlag(pi, enum(2)))
	assert.True(t, HasFlag(pi, enum(4)))

	assert.False(t, HasFlag(pi, enum(1)))
	assert.False(t, HasFlag(pi, enum(3)))
	assert.False(t, HasFlag(pi, enum(0)))
}

func TestSetFlag(t *testing.T) {
	i := enum(0)
	pi := (*int64)(&i)

	SetFlag(pi, true, enum(1))
	assert.Equal(t, enum(2), i)

	SetFlag(pi, true, enum(4), enum(7))
	assert.Equal(t, enum(146), i)

	SetFlag(pi, false, enum(4), enum(7))
	assert.Equal(t, enum(2), i)

	SetFlag(pi, false, enum(1))
	assert.Equal(t, enum(0), i)
}

func TestUnmarshal(t *testing.T) {
	i := enum(0)

	assert.NoError(t, UnmarshalText(&i, []byte("Orange"), "Fruits"))
	assert.Equal(t, enum(7), i)
	i = 4
	assert.NoError(t, UnmarshalText(&i, []byte("Apple"), "Fruits"))
	assert.Equal(t, enum(4), i)

	assert.NoError(t, UnmarshalJSON(&i, []byte(`"Orange"`), "Fruits"))
	assert.Equal(t, enum(7), i)
	i = 4
	assert.NoError(t, UnmarshalJSON(&i, []byte(`"Apple"`), "Fruits"))
	assert.Equal(t, enum(4), i)

	assert.NoError(t, UnmarshalYAML(&i, &yaml.Node{Kind: yaml.ScalarNode, Value: "Orange"}, "Fruits"))
	assert.Equal(t, enum(7), i)
	i = 4
	assert.NoError(t, UnmarshalYAML(&i, &yaml.Node{Kind: yaml.ScalarNode, Value: "Apple"}, "Fruits"))
	assert.Equal(t, enum(4), i)

	assert.NoError(t, Scan(&i, []byte("Orange"), "Fruits"))
	assert.Equal(t, enum(7), i)
	i = 4
	assert.NoError(t, Scan(&i, "Orange", "Fruits"))
	assert.Equal(t, enum(7), i)
	i = 4
	assert.NoError(t, Scan(&i, nil, "Fruits"))
	assert.Equal(t, enum(4), i)
	i = 4
	assert.Error(t, Scan(&i, enum(0), "Fruits"))
	assert.Equal(t, enum(4), i)
	i = 4
	assert.Error(t, Scan(&i, 78, "Fruits"))
	assert.Equal(t, enum(4), i)
}
