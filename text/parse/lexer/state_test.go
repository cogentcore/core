// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"testing"

	"cogentcore.org/core/parse/token"
	"github.com/stretchr/testify/assert"
)

func TestReadUntil(t *testing.T) {
	ls := &State{
		Src: []rune(" ( Hello } , ) ] Worabcld!"),
		Pos: 0,
		Ch:  'H',
	}

	ls.ReadUntil("(")
	assert.Equal(t, 2, ls.Pos)

	ls.ReadUntil("}")
	assert.Equal(t, 10, ls.Pos)

	ls.ReadUntil(")")
	assert.Equal(t, 14, ls.Pos)

	ls.ReadUntil("]")
	assert.Equal(t, 16, ls.Pos)

	ls.ReadUntil("abc")
	assert.Equal(t, 23, ls.Pos)

	ls.ReadUntil("h")
	assert.Equal(t, 26, ls.Pos)
}

func TestReadNumber(t *testing.T) {
	ls := &State{
		Src: []rune("0x1234"),
		Pos: 0,
		Ch:  '0',
	}

	tok := ls.ReadNumber()
	assert.Equal(t, token.LitNumInteger, tok)
	assert.Equal(t, 6, ls.Pos)

	ls = &State{
		Src: []rune("0123456789"),
		Pos: 0,
		Ch:  '0',
	}

	tok = ls.ReadNumber()
	assert.Equal(t, token.LitNumInteger, tok)
	assert.Equal(t, 10, ls.Pos)

	ls = &State{
		Src: []rune("3.14"),
		Pos: 0,
		Ch:  '3',
	}

	tok = ls.ReadNumber()
	assert.Equal(t, token.LitNumFloat, tok)
	assert.Equal(t, 4, ls.Pos)

	ls = &State{
		Src: []rune("1e10"),
		Pos: 0,
		Ch:  '1',
	}

	tok = ls.ReadNumber()
	assert.Equal(t, token.LitNumFloat, tok)
	assert.Equal(t, 4, ls.Pos)

	ls = &State{
		Src: []rune("42i"),
		Pos: 0,
		Ch:  '4',
	}

	tok = ls.ReadNumber()
	assert.Equal(t, token.LitNumImag, tok)
	assert.Equal(t, 3, ls.Pos)
}

func TestReadEscape(t *testing.T) {
	ls := &State{
		Src: []rune(`\n \t "hello \u03B1 \U0001F600`),
		Pos: 0,
		Ch:  '\\',
	}

	assert.True(t, ls.ReadEscape('"'))
	assert.Equal(t, 1, ls.Pos)

	assert.True(t, ls.ReadEscape('"'))
	assert.Equal(t, 2, ls.Pos)

	assert.False(t, ls.ReadEscape('"'))
	assert.Equal(t, 2, ls.Pos)

	assert.False(t, ls.ReadEscape('"'))
	assert.Equal(t, 2, ls.Pos)
}
