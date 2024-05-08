// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package indent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTabs(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, ""},
		{1, "\t"},
		{2, "\t\t"},
		{3, "\t\t\t"},
		{4, "\t\t\t\t"},
	}

	for _, test := range tests {
		result := Tabs(test.n)
		assert.Equal(t, test.expected, result)
	}
}

func TestTabBytes(t *testing.T) {
	tests := []struct {
		n        int
		expected []byte
	}{
		{0, []byte("")},
		{1, []byte("\t")},
		{2, []byte("\t\t")},
		{3, []byte("\t\t\t")},
		{4, []byte("\t\t\t\t")},
	}

	for _, test := range tests {
		result := TabBytes(test.n)
		assert.Equal(t, test.expected, result)
	}
}

func TestSpaces(t *testing.T) {
	tests := []struct {
		n        int
		width    int
		expected string
	}{
		{0, 4, ""},
		{1, 4, "    "},
		{2, 4, "        "},
		{3, 4, "            "},
		{4, 4, "                "},
	}

	for _, test := range tests {
		result := Spaces(test.n, test.width)
		assert.Equal(t, test.expected, result)
	}
}

func TestSpaceBytes(t *testing.T) {
	tests := []struct {
		n        int
		width    int
		expected []byte
	}{
		{0, 4, []byte("")},
		{1, 4, []byte("    ")},
		{2, 4, []byte("        ")},
		{3, 4, []byte("            ")},
		{4, 4, []byte("                ")},
	}

	for _, test := range tests {
		result := SpaceBytes(test.n, test.width)
		assert.Equal(t, test.expected, result)
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		ich      Char
		n        int
		width    int
		expected string
	}{
		{Tab, 0, 4, ""},
		{Tab, 1, 4, "\t"},
		{Tab, 2, 4, "\t\t"},
		{Tab, 3, 4, "\t\t\t"},
		{Tab, 4, 4, "\t\t\t\t"},
		{Space, 0, 4, ""},
		{Space, 1, 4, "    "},
		{Space, 2, 4, "        "},
		{Space, 3, 4, "            "},
		{Space, 4, 4, "                "},
	}

	for _, test := range tests {
		result := String(test.ich, test.n, test.width)
		assert.Equal(t, test.expected, result)
	}
}

func TestBytes(t *testing.T) {
	tests := []struct {
		ich      Char
		n        int
		width    int
		expected []byte
	}{
		{Tab, 0, 4, []byte("")},
		{Tab, 1, 4, []byte("\t")},
		{Tab, 2, 4, []byte("\t\t")},
		{Tab, 3, 4, []byte("\t\t\t")},
		{Tab, 4, 4, []byte("\t\t\t\t")},
		{Space, 0, 4, []byte("")},
		{Space, 1, 4, []byte("    ")},
		{Space, 2, 4, []byte("        ")},
		{Space, 3, 4, []byte("            ")},
		{Space, 4, 4, []byte("                ")},
	}

	for _, test := range tests {
		result := Bytes(test.ich, test.n, test.width)
		assert.Equal(t, test.expected, result)
	}
}

func TestLen(t *testing.T) {
	tests := []struct {
		ich      Char
		n        int
		width    int
		expected int
	}{
		{Tab, 0, 4, 0},
		{Tab, 1, 4, 1},
		{Tab, 2, 4, 2},
		{Tab, 3, 4, 3},
		{Tab, 4, 4, 4},
		{Space, 0, 4, 0},
		{Space, 1, 4, 4},
		{Space, 2, 4, 8},
		{Space, 3, 4, 12},
		{Space, 4, 4, 16},
	}

	for _, test := range tests {
		result := Len(test.ich, test.n, test.width)
		assert.Equal(t, test.expected, result)
	}
}
