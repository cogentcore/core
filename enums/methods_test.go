// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enums

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// it is much easier to test with an independent enum mock
type enum int

var hasFlag = map[enum]bool{5: true, 3: true}
var bitIndexString = map[enum]string{5: "Apple", 3: "bitIndexString", 1: "one"}

func (e enum) String() string         { return "extended" }
func (e enum) Int64() int64           { return int64(e) }
func (e enum) Desc() string           { return "" }
func (e enum) Values() []Enum         { return nil }
func (e enum) HasFlag(f BitFlag) bool { return hasFlag[f.(enum)] }
func (e enum) BitIndexString() string { return bitIndexString[e] }
func (e *enum) SetInt64(i int64)      { *e = enum(i) }
func (e *enum) SetString(s string) error {
	if s == "Orange" {
		*e = 7
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
	assert.Equal(t, "bitIndexString|Apple", BitFlagStringExtended(0, []enum{5, 1}, []enum{2, 3, 1}))
}

func TestSetString(t *testing.T) {
	nameToValueMap := map[string]enum{"apple": 5}

	i := enum(0)
	assert.NoError(t, SetString(&i, "apple", nameToValueMap, "Fruits"))
	assert.Equal(t, enum(5), i)
	i = enum(4)
	err := SetString(&i, "Apple", nameToValueMap, "Fruits")
	if assert.Error(t, err) {
		assert.Equal(t, "Apple is not a valid value for type Fruits", err.Error())
	}
	assert.Equal(t, enum(4), i)
	err = SetString(&i, "Orange", nameToValueMap, "Fruits")
	if assert.Error(t, err) {
		assert.Equal(t, "Orange is not a valid value for type Fruits", err.Error())
	}
	assert.Equal(t, enum(4), i)

	assert.NoError(t, SetStringLower(&i, "apple", nameToValueMap, "Fruits"))
	assert.Equal(t, enum(5), i)
	i = enum(4)
	assert.NoError(t, SetStringLower(&i, "Apple", nameToValueMap, "Fruits"))
	assert.Equal(t, enum(5), i)
	i = enum(4)
	err = SetStringLower(&i, "Orange", nameToValueMap, "Fruits")
	if assert.Error(t, err) {
		assert.Equal(t, "Orange is not a valid value for type Fruits", err.Error())
	}
	assert.Equal(t, enum(4), i)

	assert.NoError(t, SetStringExtended(&i, &i, "apple", nameToValueMap))
	assert.Equal(t, enum(5), i)
	i = enum(4)
	assert.NoError(t, SetStringExtended(&i, &i, "Orange", nameToValueMap))
	assert.Equal(t, enum(7), i)
	i = enum(4)
	err = SetStringExtended(&i, &i, "Apple", nameToValueMap)
	if assert.Error(t, err) {
		assert.Equal(t, "invalid", err.Error())
	}
	assert.Equal(t, enum(4), i)

	assert.NoError(t, SetStringLowerExtended(&i, &i, "apple", nameToValueMap))
	assert.Equal(t, enum(5), i)
	i = enum(4)
	assert.NoError(t, SetStringLowerExtended(&i, &i, "Apple", nameToValueMap))
	assert.Equal(t, enum(5), i)
	i = enum(4)
	assert.NoError(t, SetStringLowerExtended(&i, &i, "Orange", nameToValueMap))
	assert.Equal(t, enum(7), i)
	i = enum(4)
	err = SetStringLowerExtended(&i, &i, "Strawberry", nameToValueMap)
	if assert.Error(t, err) {
		assert.Equal(t, "invalid", err.Error())
	}
	assert.Equal(t, enum(4), i)
}
