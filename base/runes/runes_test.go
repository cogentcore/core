// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqualFold(t *testing.T) {
	tests := []struct {
		s        []rune
		t        []rune
		expected bool
	}{
		{[]rune("hello"), []rune("hello"), true},
		{[]rune("Hello"), []rune("hello"), true},
		{[]rune("hello"), []rune("HELLO"), true},
		{[]rune("world"), []rune("word"), false},
		{[]rune("abc"), []rune("def"), false},
		{[]rune(""), []rune(""), true},
		{[]rune("abc"), []rune(""), false},
		{[]rune(""), []rune("def"), false},
	}

	for _, test := range tests {
		result := EqualFold(test.s, test.t)
		assert.Equal(t, test.expected, result)
	}
}

func TestIndex(t *testing.T) {
	tests := []struct {
		txt      []rune
		find     []rune
		expected int
	}{
		{[]rune("hello"), []rune("el"), 1},
		{[]rune("Hello"), []rune("l"), 2},
		{[]rune("world"), []rune("or"), 1},
		{[]rune("abc"), []rune("def"), -1},
		{[]rune(""), []rune("def"), -1},
		{[]rune("abc"), []rune(""), -1},
		{[]rune(""), []rune(""), -1},
	}

	for _, test := range tests {
		result := Index(test.txt, test.find)
		assert.Equal(t, test.expected, result)
	}
}

func TestIndexFold(t *testing.T) {
	tests := []struct {
		txt      []rune
		find     []rune
		expected int
	}{
		{[]rune("hello"), []rune("el"), 1},
		{[]rune("Hello"), []rune("l"), 2},
		{[]rune("world"), []rune("or"), 1},
		{[]rune("abc"), []rune("def"), -1},
		{[]rune(""), []rune("def"), -1},
		{[]rune("abc"), []rune(""), -1},
		{[]rune(""), []rune(""), -1},
		{[]rune("hello"), []rune("EL"), 1},
		{[]rune("Hello"), []rune("L"), 2},
		{[]rune("world"), []rune("OR"), 1},
		{[]rune("abc"), []rune("DEF"), -1},
		{[]rune(""), []rune("DEF"), -1},
		{[]rune("abc"), []rune(""), -1},
		{[]rune(""), []rune(""), -1},
	}

	for _, test := range tests {
		result := IndexFold(test.txt, test.find)
		assert.Equal(t, test.expected, result)
	}
}

func TestRepeat(t *testing.T) {
	tests := []struct {
		r        []rune
		count    int
		expected []rune
	}{
		{[]rune("hello"), 0, []rune{}},
		{[]rune("hello"), 1, []rune("hello")},
		{[]rune("hello"), 2, []rune("hellohello")},
		{[]rune("world"), 3, []rune("worldworldworld")},
		{[]rune(""), 5, []rune("")},
	}

	for _, test := range tests {
		result := Repeat(test.r, test.count)
		assert.Equal(t, test.expected, result)
	}
}
