// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/goscript"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/text/highlighting/lexers"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/token"
	"github.com/alecthomas/chroma/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStyleEntry tests the string, CSS, properties, and differences of
// style entries.
func TestStyleEntry(t *testing.T) {
	se := StyleEntry{Color: colors.Red, Background: colors.White, Border: colors.Blue, Bold: Yes, Italic: No, DottedUnderline: Yes, NoInherit: true}
	se.UpdateFromTheme()
	s := se.String()
	assert.Contains(t, s, "bold")
	assert.Contains(t, s, "noitalic")
	assert.Contains(t, s, "dotted-underline", "the dotted underline, not the underline")
	assert.Contains(t, s, "noinherit")
	assert.Contains(t, s, "bg:")
	assert.Contains(t, s, "border:")
	assert.Equal(t, "", Pass.Prefix("x"))

	css := se.ToCSS()
	assert.Contains(t, css, "color: ")
	assert.Contains(t, css, "background-color: ")
	assert.Contains(t, css, "font-weight: bold")
	assert.Contains(t, css, "text-decoration: dotted-underline")
	assert.Contains(t, StyleEntry{Underline: Yes, Italic: Yes}.ToCSS(), "text-decoration: underline")
	assert.Contains(t, StyleEntry{Italic: Yes}.ToCSS(), "font-style: italic")

	pr := se.ToProperties()
	assert.Equal(t, 1<<uint32(rich.DottedUnderline), pr["text-decoration"], "dotted")
	assert.Equal(t, rich.Bold, pr["font-weight"])
	assert.Equal(t, 1<<uint32(rich.Underline), StyleEntry{Underline: Yes}.ToProperties()["text-decoration"])
	assert.Equal(t, rich.Italic, StyleEntry{Italic: Yes}.ToProperties()["font-style"])

	sty := rich.NewStyle()
	StyleEntry{DottedUnderline: Yes, Italic: Yes}.ToRichStyle(sty)
	assert.True(t, sty.Decoration.HasFlag(rich.DottedUnderline))
	assert.Equal(t, rich.Italic, sty.Slant)

	o := StyleEntry{Color: colors.Red, Bold: Yes}
	d := se.Sub(o)
	assert.True(t, colors.IsNil(d.Color), "the same color")
	assert.Equal(t, Pass, d.Bold, "the same weight")
	assert.Equal(t, colors.White, d.Background)
	assert.Equal(t, colors.Blue, d.Border)
	assert.Equal(t, No, d.Italic)
	assert.Equal(t, Yes, d.DottedUnderline)
	assert.Equal(t, Pass, StyleEntry{Underline: Yes}.Sub(StyleEntry{Underline: Yes}).Underline)
	assert.Equal(t, No, StyleEntry{Underline: No}.Sub(StyleEntry{}).Underline)
}

// TestStyleFiles tests the CSS of a style, and saving and opening styles.
func TestStyleFiles(t *testing.T) {
	goscript.Skip(t, "temporary directories")
	st := AvailableStyle(DefaultStyle)
	require.NotNil(t, st)
	css := st.ToCSS()
	assert.NotEmpty(t, css[token.Comment])

	dir := t.TempDir()
	fn := fsx.Filename(filepath.Join(dir, "style.highlighting"))
	require.NoError(t, st.SaveJSON(fn))
	var got Style
	require.NoError(t, got.OpenJSON(fn), "into an empty style")
	assert.Equal(t, len(*st), len(got))
	assert.Error(t, got.OpenJSON(fsx.Filename(filepath.Join(dir, "missing"))))
	assert.Error(t, st.SaveJSON(fsx.Filename(filepath.Join(dir, "no", "dir", "x"))))

	hs := Styles{}
	added := hs.Add()
	assert.NotNil(t, added)
	assert.Contains(t, hs, "NewStyle_0")
	hs["mine"] = st
	sfn := fsx.Filename(filepath.Join(dir, "styles.json"))
	require.NoError(t, hs.SaveJSON(sfn))
	var hs2 Styles
	require.NoError(t, hs2.OpenJSON(sfn))
	assert.Len(t, hs2, 2)
	hs.SaveAll(fsx.Filename(dir))
	assert.FileExists(t, filepath.Join(dir, "mine.highlighting"))
	assert.Error(t, hs.SaveJSON(fsx.Filename(filepath.Join(dir, "no", "dir", "x"))))
}

// TestChromaMarkup tests the tags of a whole text and of a line with a
// chroma lexer, and a highlighter that is off.
func TestChromaMarkup(t *testing.T) {
	goscript.Skip(t, "HTML escaping")
	AddLexers(lexers.TOML)
	fi, err := fileinfo.NewFileInfo("x.toml")
	_ = err
	var pst parse.FileStates
	pst.SetSrc("x.toml", "", fi.Known)
	hi := Highlighter{}
	hi.Init(fi, &pst)
	require.True(t, hi.Has)
	assert.Equal(t, "TOML", hi.language)
	src := "# comment\nkey = \"value\"\n"
	all, err := hi.MarkupTagsAll([]byte(src))
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.NotEmpty(t, all[1])
	ln, err := hi.MarkupTagsLine(1, []rune("key = \"value\""))
	require.NoError(t, err)
	assert.Equal(t, all[1], ln)

	off := Highlighter{off: true}
	tags, err := off.MarkupTagsAll([]byte(src))
	assert.Nil(t, tags)
	assert.NoError(t, err)
	line, _ := off.MarkupTagsLine(0, []rune(src))
	assert.Nil(t, line)
	none := Highlighter{}
	tags, _ = none.MarkupTagsAll([]byte(src))
	assert.Nil(t, tags)
	line, _ = none.MarkupTagsLine(0, []rune(src))
	assert.Nil(t, line)

	assert.Equal(t, "a &lt;b&gt; &amp; &#39;c&#39;", string(HTMLEscapeBytes([]byte("a <b> & 'c'"))))
	assert.Equal(t, "&#34;x&#34;", string(HTMLEscapeRunes([]rune(`"x"`))))
	assert.Equal(t, chroma.Comment, TokenToChroma(token.Comment))
	assert.Equal(t, token.Comment, TokenFromChroma(TokenToChroma(token.Comment)))
}
