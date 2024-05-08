// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package option

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	optInt := New(10)
	assert.NotNil(t, optInt)
	assert.Equal(t, 10, optInt.Value)

	optStr := New("Hello, World!")
	assert.NotNil(t, optStr)
	assert.Equal(t, "Hello, World!", optStr.Value)

	type Person struct {
		Name string
		Age  int
	}
	optStruct := New(Person{Name: "John Doe", Age: 30})
	assert.NotNil(t, optStruct)
	assert.Equal(t, Person{Name: "John Doe", Age: 30}, optStruct.Value)
}

func TestSet(t *testing.T) {
	opt := New(0)
	assert.NotNil(t, opt)
	assert.Equal(t, 0, opt.Value)
	assert.True(t, opt.Valid)

	opt.Set(42)
	assert.Equal(t, 42, opt.Value)
	assert.True(t, opt.Valid)

	optStr := New("")
	optStr.Set("Hello, World!")
	assert.Equal(t, "Hello, World!", optStr.Value)
	assert.True(t, optStr.Valid)

	type Person struct {
		Name string
		Age  int
	}
	optStruct := New(Person{})
	assert.NotNil(t, optStruct)
	assert.Equal(t, Person{}, optStruct.Value)
	assert.True(t, optStruct.Valid)

	optStruct.Set(Person{Name: "John Doe", Age: 30})
	assert.Equal(t, Person{Name: "John Doe", Age: 30}, optStruct.Value)
	assert.True(t, optStruct.Valid)
}
