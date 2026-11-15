// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"testing"

	"cogentcore.org/core/base/indent"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
)

// TestStringHelpers tests the helpers for strings used in completion.
func TestStringHelpers(t *testing.T) {
	assert.Equal(t, "abc", TrimLeftToAlpha("(*[abc"))
	assert.Equal(t, "", TrimLeftToAlpha("123"))
	assert.Equal(t, 2, FirstNonSpaceRune([]rune("  x ")))
	assert.Equal(t, -1, FirstNonSpaceRune([]rune("   ")))
	assert.Equal(t, 2, LastNonSpaceRune([]rune("  x ")))
	assert.Equal(t, -1, LastNonSpaceRune([]rune("  ")))
	assert.Equal(t, -1, LastNonSpaceRune(nil))
	assert.Equal(t, "c.d", LastField("a b c.d"))
	assert.Equal(t, "", LastField(""))

	assert.Equal(t, "fmt.Pri", LastScopedString("x := f(fmt.Pri"))
	assert.Equal(t, "y", LastScopedString("f(x, y"))
	assert.Equal(t, "b", LastScopedString("m[a][b"))
	assert.Equal(t, "{", LastScopedString("{"), "only a bracket")
	assert.Equal(t, "x", LastScopedString("x)"))
	assert.Equal(t, "(1", LastScopedString("(1"), "no letters")

	assert.True(t, HasUpperCase("abC"))
	assert.False(t, HasUpperCase("abc"))
	assert.Equal(t, "FooBar", MatchCase("XyzAbc", "foobar"))
	assert.Equal(t, "fOObar", MatchCase("aBC", "FoObar"))
	assert.Equal(t, "Ab", MatchCase("Abcdef", "ab"))
}

// TestLineHelpers tests the methods of lines of lexical elements.
func TestLineHelpers(t *testing.T) {
	src := []rune("var x = 1 // a comment")
	ln := Line{
		NewLex(token.KeyToken{Token: token.Keyword}, 0, 3),
		NewLex(token.KeyToken{Token: token.Name}, 4, 5),
		NewLex(token.KeyToken{Token: token.OpAsgnAssign}, 6, 7),
		NewLex(token.KeyToken{Token: token.LitNumInteger}, 8, 9),
		NewLex(token.KeyToken{Token: token.Comment}, 10, 22),
	}
	assert.Equal(t, []string{"var", "x", "=", "1", "// a comment"}, ln.Strings(src))
	assert.Equal(t, ln.Strings(src), ln.RuneStrings(src))
	assert.Nil(t, (&Line{}).Strings(src))
	assert.Contains(t, ln.String(), "Keyword")
	assert.Contains(t, ln.TagSrc(src), `"var"`)

	words := ln.NonCodeWords(src)
	assert.Equal(t, []string{"1", "//", "a", "comment"}, words.Strings(src), "not the code: literals are not code")
	assert.Nil(t, RuneFields(nil))
	f := RuneFields([]rune(" ab  cd"))
	assert.Equal(t, []string{"ab", "cd"}, f.Strings([]rune(" ab  cd")))

	cp := ln.Clone()
	cp.DeleteToken(token.Comment)
	assert.Len(t, cp, 4)
	cp.DeleteIndex(0)
	assert.Equal(t, token.Name, cp[0].Token.Token)
	assert.Len(t, ln, 5, "the original is not changed")

	un := Line{ln[3], ln[0], NewLex(token.KeyToken{Token: token.Name}, 0, 1)}
	un.Sort()
	assert.Equal(t, 0, un[0].Start)
	assert.Equal(t, 3, un[0].End, "longer first at the same start")
	assert.Equal(t, 1, un[1].End)
	assert.Equal(t, 8, un[2].Start)

	lx := ln[1]
	assert.Equal(t, textpos.Region{Start: textpos.Pos{Line: 2, Char: 4}, End: textpos.Pos{Line: 2, Char: 5}}, lx.Region(2))
	assert.True(t, lx.OverlapsReg(NewLex(token.KeyToken{}, 3, 5)))
	assert.True(t, lx.OverlapsReg(NewLex(token.KeyToken{}, 4, 9)))
	assert.True(t, lx.OverlapsReg(NewLex(token.KeyToken{}, 0, 9)), "containing")
	assert.False(t, lx.OverlapsReg(NewLex(token.KeyToken{}, 5, 9)))
	assert.False(t, lx.OverlapsReg(NewLex(token.KeyToken{}, 0, 4)))
	lx.Now()
	assert.False(t, lx.Time.IsZero())

	// object paths
	psrc := []rune("x := a.b.c")
	pl := Line{
		NewLex(token.KeyToken{Token: token.Name}, 0, 1),
		NewLex(token.KeyToken{Token: token.OpAsgnDefine}, 2, 4),
		NewLex(token.KeyToken{Token: token.Name}, 5, 6),
		NewLex(token.KeyToken{Token: token.PunctSepPeriod}, 6, 7),
		NewLex(token.KeyToken{Token: token.Name}, 7, 8),
		NewLex(token.KeyToken{Token: token.PunctSepPeriod}, 8, 9),
		NewLex(token.KeyToken{Token: token.Name}, 9, 10),
	}
	st := ObjPathAt(pl, &pl[6])
	assert.Equal(t, "a", string(st.Src(psrc)))
	assert.Same(t, &pl[0], ObjPathAt(pl, &pl[0]), "at the start")
}

// TestPosStack tests the eos positions and the state stack.
func TestPosStack(t *testing.T) {
	ep := EosPos{2, 5, 9}
	assert.Equal(t, 5, ep.FindGt(2))
	assert.Equal(t, -1, ep.FindGt(9))
	assert.Equal(t, 2, ep.FindGtEq(2))

	var st Stack
	assert.Equal(t, "", st.Top())
	assert.Equal(t, "", st.Pop())
	st.Push("a")
	st.Push("b")
	assert.Equal(t, "b", st.Top())
	assert.Equal(t, "b", st.Pop())
	assert.Equal(t, "a", st.Top())
}

// TestIndentHelpers tests the indentation of lines after brackets.
func TestIndentHelpers(t *testing.T) {
	src := [][]rune{[]rune("f(x) {"), []rune(""), []rune("\tx"), []rune("}"), []rune("\ty // c")}
	tags := []Line{
		{NewLex(token.KeyToken{Token: token.Name}, 0, 1), NewLex(token.KeyToken{Token: token.PunctGpLParen}, 1, 2), NewLex(token.KeyToken{Token: token.Name}, 2, 3), NewLex(token.KeyToken{Token: token.PunctGpRParen}, 3, 4), NewLex(token.KeyToken{Token: token.PunctGpLBrace}, 5, 6)},
		{},
		{NewLex(token.KeyToken{Token: token.Name}, 1, 2)},
		{NewLex(token.KeyToken{Token: token.PunctGpRBrace}, 0, 1)},
		{NewLex(token.KeyToken{Token: token.Name}, 1, 2), NewLex(token.KeyToken{Token: token.Comment}, 3, 7)},
	}
	st, end := LineStartEndBracket(src[0], tags[0])
	assert.False(t, st)
	assert.True(t, end)
	st, end = LineStartEndBracket(src[3], tags[3])
	assert.True(t, st)
	assert.False(t, end)
	st, end = LineStartEndBracket(nil, nil)
	assert.False(t, st || end)
	ll, li := LastLexIgnoreComment(tags[4])
	assert.Equal(t, 0, li)
	assert.Equal(t, token.Name, ll.Token.Token)

	ind, pln, ich := PrevLineIndent(src, tags, 3, 4)
	assert.Equal(t, 1, ind)
	assert.Equal(t, 2, pln)
	assert.Equal(t, indent.Tab, ich)
	ind, pln, _ = PrevLineIndent(src, tags, 2, 4)
	assert.Equal(t, 0, ind, "skips the blank line")
	assert.Equal(t, 0, pln)
	ind, pln, _ = PrevLineIndent(src, tags, 0, 4)
	assert.Equal(t, 0, ind+pln)

	_, del, _, _ := BracketIndentLine(src, tags, 2, 4)
	assert.Equal(t, 1, del, "after an opening brace")
	_, del, _, _ = BracketIndentLine(src, tags, 3, 4)
	assert.Equal(t, -1, del, "a closing brace")
	_, del, _, _ = BracketIndentLine(src, tags, 4, 4)
	assert.Equal(t, 0, del, "no undent below zero")
	_, del, _, _ = BracketIndentLine(src, tags, 0, 4)
	assert.Equal(t, 0, del)
}
