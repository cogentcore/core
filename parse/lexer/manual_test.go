// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirstWord(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello, World!", "Hello"},
		{"This is a test", "This"},
		{"12345", ""},
		{"!@#$%", ""},
		{"", ""},
	}

	for _, test := range tests {
		result := FirstWord(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestFirstWordDigits(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello, World!", "Hello"},
		{"This is a test", "This"},
		{"12345", ""},
		{"!@#$%", ""},
		{"", ""},
		{"123abc", "abc"},
		{"abc123", "abc123"},
		{"abc123def", "abc123def"},
		{"123abc456", "abc456"},
	}

	for _, test := range tests {
		result := FirstWordDigits(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestFirstWordApostrophe(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello, World!", "Hello"},
		{"This is a test", "This"},
		{"12345", ""},
		{"!@#$%", ""},
		{"", ""},
		{"'Hello'", "Hello"},
		{"'This is a test'", "This"},
		{"'12345'", ""},
		{"'!@#$%'", ""},
		{"''", ""},
		{"'Hello", "Hello"},
		{"'Hello'", "Hello"},
		{"'Hello''", "Hello"},
		{"'Hello'''", "Hello"},
	}

	for _, test := range tests {
		result := FirstWordApostrophe(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestMarkupPathsAsLinks(t *testing.T) {
	flds := []string{
		"./path/file.go",
		"/absolute/path/file.go",
		"../relative/path/file.go",
		"file.go",
	}

	orig, link := MarkupPathsAsLinks(flds, 3)

	expectedOrig := []byte("./path/file.go")
	expectedLink := []byte(`<a href="file:///./path/file.go">./path/file.go</a>`)

	assert.Equal(t, expectedOrig, orig)
	assert.Equal(t, expectedLink, link)
}
