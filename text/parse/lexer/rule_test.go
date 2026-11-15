// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuleLexStart(t *testing.T) {
	rule := &Rule{
		Match:  String,
		String: "Hello",
		Acts:   []Actions{Next},
	}
	state := &State{
		Src: []rune("Hello, World!"),
	}

	rule.LexStart(state)

	assert.Equal(t, 0, state.Pos)
}

func TestRuleLex(t *testing.T) {
	rule := &Rule{
		Match:  String,
		String: "Hello",
		Acts:   []Actions{Next},
	}
	state := &State{
		Src: []rune("Hello, World!"),
	}

	rule.Lex(state)

	assert.Equal(t, 0, state.Pos)
}

// TestIsMatchPos tests the positions in a line at which rules match.
func TestIsMatchPos(t *testing.T) {
	src := []rune("ab cd ef")
	match := func(pos MatchPos, str string, at int) bool {
		lr := &Rule{Match: String, String: str, Pos: pos}
		return lr.IsMatchPos(&State{Src: src, Pos: at})
	}
	assert.True(t, match(AnyPos, "cd", 3))
	assert.True(t, match(StartOfLine, "ab", 0))
	assert.False(t, match(StartOfLine, "cd", 3))
	assert.True(t, match(EndOfLine, "ef", 6))
	assert.False(t, match(EndOfLine, "cd", 3))
	assert.True(t, match(MiddleOfLine, "cd", 3))
	assert.False(t, match(MiddleOfLine, "ab", 0))
	assert.False(t, match(MiddleOfLine, "ef", 6))
	assert.True(t, match(StartOfWord, "cd", 3))
	assert.False(t, match(StartOfWord, "d", 4))
	assert.True(t, match(EndOfWord, "cd", 3))
	assert.True(t, match(EndOfWord, "d", 4))
	assert.True(t, match(EndOfWord, "ef", 6), "at the end of the line")
	assert.False(t, match(EndOfWord, "c", 3))
	assert.False(t, match(MiddleOfWord, "b", 1), "b ends its word")
	assert.False(t, match(MiddleOfWord, "c", 3), "c starts its word")
	assert.True(t, (&Rule{Match: String, String: "x", Pos: MiddleOfWord}).IsMatchPos(&State{Src: []rune("axb"), Pos: 1}))

	for _, m := range []Matches{Letter, Digit, WhiteSpace, AnyRune} {
		assert.Equal(t, 2, (&Rule{Match: m, Offset: 1}).TargetLen(nil), m.String())
	}
	assert.Equal(t, 0, (&Rule{Match: CurState}).TargetLen(nil))
	assert.Equal(t, 3, (&Rule{Match: StrName, String: "abc"}).TargetLen(nil))
}
