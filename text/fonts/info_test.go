// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fonts

import (
	iofs "io/fs"
	"os"
	"testing"
	"testing/fstest"

	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"github.com/cogentcore/typesetting/fontscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFamilies tests the families of a list of fonts.
func TestFamilies(t *testing.T) {
	fi := []Info{{Family: "B"}, {Family: "A", Weight: rich.Bold}, {Family: "A"}, {Family: "C"}, {Family: "B"}}
	ff := Families(fi)
	require.Len(t, ff, 3)
	assert.Equal(t, "A", ff[0].Label())
	assert.Equal(t, "B", ff[1].Family)
	assert.Equal(t, "C", ff[2].Family)
	assert.Equal(t, InfoExample, ff[0].Example)
	assert.Equal(t, "A", fi[0].Label(), "sorted")
	assert.Empty(t, Families(nil))
}

// TestDataStyle tests loading font data, and styling text with a face.
func TestDataStyle(t *testing.T) {
	b, err := os.ReadFile("noto/NotoSans-BoldItalic.ttf")
	require.NoError(t, err)
	fd := &Data{Family: "Noto Sans", Data: b}
	require.NoError(t, fd.Load())
	require.Len(t, fd.Fonts, 1)

	sty := rich.NewStyle()
	tsty := text.NewStyle()
	Style(fd.Fonts[0], sty, tsty)
	assert.Equal(t, rich.FontName("Noto Sans"), tsty.CustomFont)
	assert.Equal(t, rich.Custom, sty.Family)
	assert.Equal(t, rich.Bold, sty.Weight)
	assert.Equal(t, rich.Italic, sty.Slant)
	Style(nil, sty, tsty) // no change

	bad := &Data{Data: []byte("not a font")}
	assert.Error(t, bad.Load())
}

// TestUseInMap tests adding the fonts of file systems to a font map.
func TestUseInMap(t *testing.T) {
	b, err := os.ReadFile("noto/NotoSans-Regular.ttf")
	require.NoError(t, err)
	fm := fontscan.NewFontMap(nil)
	assert.NoError(t, UseInMap(fm, []iofs.FS{fstest.MapFS{"fonts/a.ttf": {Data: b}}}))
	assert.Error(t, UseInMap(fm, []iofs.FS{fstest.MapFS{"bad.ttf": {Data: []byte("x")}}}))

	n := len(Embedded)
	AddEmbedded(fstest.MapFS{"a.ttf": {Data: b}})
	assert.Len(t, Embedded, n+1)
	assert.NoError(t, UseEmbeddedInMap(fontscan.NewFontMap(nil)))
	Embedded = Embedded[:n]
}

// TestWebFonts tests adding web fonts and getting the new ones.
func TestWebFonts(t *testing.T) {
	b, err := os.ReadFile("noto/NotoSans-Italic.ttf")
	require.NoError(t, err)
	_, n := WebFonts(0)
	assert.Error(t, AddWebFont("Bad", []byte("not a font")))
	assert.False(t, HasWebFont("bad"))
	require.NoError(t, AddWebFont("My Font", b))
	assert.True(t, HasWebFont("my font"), "any case")
	nw, total := WebFonts(n)
	assert.Equal(t, n+1, total)
	require.Len(t, nw, 1)
	assert.Equal(t, "My Font", nw[0].Family)
	nw, _ = WebFonts(total)
	assert.Nil(t, nw, "no new fonts")
}
