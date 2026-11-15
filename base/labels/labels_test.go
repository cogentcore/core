// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package labels

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type labeled struct{ Name string }

func (l *labeled) Label() string { return "label " + l.Name }

type myStruct struct{ A int }

func TestFriendlyTypeName(t *testing.T) {
	tests := []struct {
		v    any
		want string
	}{
		{"", "Text"},
		{3, "Number"},
		{float32(1), "Number"},
		{&myStruct{}, "My struct"},
		{[]int{}, "Numbers"},
		{[]string{}, "Texts"},
		{[][]int{}, "List of Numbers"},
		{map[string]float64{}, "Numbers"},
		{func() {}, "Function"},
		{func(int) {}, "Function of Number"},
		{func(int, string) {}, "Function of Number and Text"},
		{func(int, string, bool) {}, "Function of Number, Text, and Bool"},
		{[]func(int){}, "Functions of Numbers"},
		{[]any{}, "Values"},
		{struct{}{}, "struct {}"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, FriendlyTypeName(reflect.TypeOf(tt.v)), "%T", tt.v)
	}
	assert.Equal(t, "None", FriendlyTypeName(nil))
}

func TestFriendlyLabels(t *testing.T) {
	assert.Equal(t, "label x", FriendlyStructLabel(reflect.ValueOf(&labeled{Name: "x"})))
	assert.Equal(t, "My struct", FriendlyStructLabel(reflect.ValueOf(&myStruct{A: 1})))
	assert.Equal(t, "None", FriendlyStructLabel(reflect.ValueOf((*myStruct)(nil))))
	assert.Equal(t, "None", FriendlyStructLabel(reflect.Value{}))

	assert.Equal(t, "3 numbers", FriendlySliceLabel(reflect.ValueOf([]int{1, 2, 3})))
	assert.Equal(t, "2 numbers", FriendlySliceLabel(reflect.ValueOf(&[2]int{})))
	assert.Equal(t, "1 lists of numbers", FriendlySliceLabel(reflect.ValueOf([][]int{{1}})))
	assert.Equal(t, "None", FriendlySliceLabel(reflect.ValueOf([]int(nil))))
	assert.Equal(t, "None", FriendlySliceLabel(reflect.Value{}))

	assert.Equal(t, "2 texts", FriendlyMapLabel(reflect.ValueOf(map[int]string{1: "a", 2: "b"})))
	assert.Equal(t, "1 lists of numbers", FriendlyMapLabel(reflect.ValueOf(map[string][]int{"a": nil})))
	assert.Equal(t, "None", FriendlyMapLabel(reflect.ValueOf(map[string]int(nil))))
}

func TestToLabel(t *testing.T) {
	assert.Equal(t, "label y", ToLabel(&labeled{Name: "y"}))
	assert.Equal(t, "3", ToLabel(3))
	lb, ok := ToLabeler(&labeled{Name: "z"})
	assert.True(t, ok)
	assert.Equal(t, "label z", lb)
	_, ok = ToLabeler(3)
	assert.False(t, ok)
}
