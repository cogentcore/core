// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/ettle/strcase
// Copyright (c) 2020 Liyan David Chang under the MIT License

package strcase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsUnicodeType(t *testing.T) {
	lowers := []rune{
		'c',
		'ҥ',
		'ȃ',
		'ñ',
		'γ',
	}
	uppers := []rune{
		'C',
		'Ҥ',
		'Ȃ',
		'Ñ',
		'Γ',
	}
	numbers := []rune{
		'6',
		'³',
		'０',
	}
	spaces := []rune{
		' ',
		'\t',
		'\n',
		'\r',
		8287, // medium mathematical space
	}
	others := []rune{
		0,
		'.',
		'_',

		8203, // zero width space doesn't have unicode white space property
	}

	t.Run("uppercase", func(t *testing.T) {
		for _, r := range uppers {
			t.Run(string(r), func(t *testing.T) {
				assert.True(t, isUpper(r))
				assert.True(t, !isLower(r))
				assert.True(t, !isNumber(r))
				assert.True(t, !isSpace(r))
			})
		}
	})
	t.Run("lowercase", func(t *testing.T) {
		for _, r := range lowers {
			t.Run(string(r), func(t *testing.T) {
				assert.True(t, !isUpper(r))
				assert.True(t, isLower(r))
				assert.True(t, !isNumber(r))
				assert.True(t, !isSpace(r))
			})
		}
	})
	t.Run("numbers", func(t *testing.T) {
		for _, r := range numbers {
			t.Run(string(r), func(t *testing.T) {
				assert.True(t, !isUpper(r))
				assert.True(t, !isLower(r))
				assert.True(t, isNumber(r))
				assert.True(t, !isSpace(r))
			})
		}
	})
	t.Run("spaces", func(t *testing.T) {
		for _, r := range spaces {
			t.Run(string(r), func(t *testing.T) {
				assert.True(t, !isUpper(r))
				assert.True(t, !isLower(r))
				assert.True(t, !isNumber(r))
				assert.True(t, isSpace(r))
			})
		}
	})
	t.Run("other", func(t *testing.T) {
		for _, r := range others {
			t.Run(string(r), func(t *testing.T) {
				assert.True(t, !isUpper(r))
				assert.True(t, !isLower(r))
				assert.True(t, !isNumber(r))
				assert.True(t, !isSpace(r))
			})
		}
	})
}

func TestToUpper(t *testing.T) {
	tests := []struct {
		r    rune
		want rune
	}{
		{'c', 'C'},
		{'A', 'A'},
		{'ñ', 'Ñ'},
		{'9', '9'},
		{'.', '.'},
	}
	for _, test := range tests {
		t.Run(string(test.r), func(t *testing.T) {
			assert.True(t, test.want == toUpper(test.r))
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		r    rune
		want rune
	}{
		{'C', 'c'},
		{'h', 'h'},
		{'Ñ', 'ñ'},
		{'9', '9'},
		{'.', '.'},
	}
	for _, test := range tests {
		t.Run(string(test.r), func(t *testing.T) {
			assert.True(t, test.want == toLower(test.r))
		})
	}
}
