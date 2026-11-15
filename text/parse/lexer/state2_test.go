// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// escape returns a state at the start of the given escape sequence (after
// its backslash).
func escape(s string) *State {
	src := []rune(s)
	ls := &State{Src: src}
	if len(src) > 0 {
		ls.Rune = src[0]
	} else {
		ls.Rune = -1
	}
	return ls
}

// TestReadEscapeKinds tests reading octal, hex, and unicode escapes, and
// the errors of unknown, unterminated, illegal, and invalid ones.
func TestReadEscapeKinds(t *testing.T) {
	for _, s := range []string{"101", "x41", "u03B1", "U0001F600", `\`, "a"} {
		ls := escape(s)
		assert.True(t, ls.ReadEscape('"'), s)
		assert.Empty(t, ls.Errs, s)
	}
	ls := escape("101z")
	ls.ReadEscape('"')
	assert.Equal(t, 3, ls.Pos, "three octal digits")

	tests := []struct{ src, msg string }{
		{"q", "unknown escape sequence"},
		{"", "escape sequence not terminated"},
		{"xZ1", "illegal character"},
		{"uD800", "invalid Unicode code point"},
		{"777", "invalid Unicode code point"}, // more than 255
	}
	for _, tt := range tests {
		ls := escape(tt.src)
		assert.False(t, ls.ReadEscape('"'), tt.src)
		if assert.Len(t, ls.Errs, 1, tt.src) {
			assert.Contains(t, ls.Errs[0].Msg, tt.msg, tt.src)
		}
	}
	ls = escape("x4")
	ls.Src = ls.Src[:2]
	assert.False(t, ls.ReadEscape('"'), "too few digits")
}

// TestStateHelpers tests the stack of states, the rest of the line, the
// tagged line, and strings at offsets.
func TestStateHelpers(t *testing.T) {
	ls := &State{}
	ls.Init()
	ls.SetLine([]rune("hello world"))
	ls.PushState("Code")
	ls.PushState("Inner")
	assert.Equal(t, "Inner", ls.CurState())
	assert.True(t, ls.MatchState("Inner"))
	assert.Equal(t, "Inner", ls.PopState())
	assert.Equal(t, "Code", ls.CurState())

	assert.Equal(t, "hello world", ls.NextSrcLine())
	ls.Pos = 6
	assert.Equal(t, "world", ls.NextSrcLine())
	s, ok := ls.String(0, 5)
	assert.True(t, ok)
	assert.Equal(t, "world", s)
	_, ok = ls.String(0, 10)
	assert.False(t, ok, "out of range")
	assert.Equal(t, "w", ls.ReadNameTmp(0)[:1])
	ls.Pos = len(ls.Src)
	assert.Equal(t, "EOL", ls.NextSrcLine())
	assert.Contains(t, ls.LineString(), "[0,11]")

	pt := &PassTwo{}
	ts := &TwoState{}
	assert.False(t, pt.HasErrs(ts))
	ts.Errs.Add(ts.Pos, "x", "err", "", nil)
	assert.True(t, pt.HasErrs(ts))
}
