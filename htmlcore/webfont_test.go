// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/goscript"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/fonts"
	"cogentcore.org/core/text/fonts/robotomono"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebFonts tests that the web fonts of @font-face rules are loaded,
// for Context.WebFonts, and used for their families (as the first family
// of font-family that is loaded), and not those of other scripts.
func TestWebFonts(t *testing.T) {
	goscript.Skip(t, "the js shaper uses the fonts of the browser")
	data, err := robotomono.Embedded.ReadFile("RobotoMono-Regular.ttf")
	require.NoError(t, err)
	file := filepath.Join(t.TempDir(), "web.ttf")
	require.NoError(t, os.WriteFile(file, data, 0666))
	assert.Equal(t, []string{"url(a.woff2) format(\"woff2\")", "local(x)"}, splitTopLevel(`url(a.woff2) format("woff2"), local(x)`, ','))
	assert.True(t, unicodeRangeHas("U+0000-00FF, U+0131", 'A'))
	assert.False(t, unicodeRangeHas("U+0400-045F", 'A'))

	b := core.NewBody()
	ctx := NewContext()
	ctx.WebFonts = true
	assert.NoError(t, ReadHTMLString(ctx, b, `<style>`+
		`@font-face { font-family: "Test Web Mono"; src: local("x"), url(file://`+file+`) format("truetype") }`+
		`@font-face { font-family: "Test Web Cyrillic"; src: url(file://`+file+`); unicode-range: U+0400-045F }`+
		`.web { font-family: "Test Web Mono", serif } .plain { font-family: serif } .short { font: 16px "Test Web Mono" }</style>`+
		`<p class="web">iiiiiiii</p><p class="plain">iiiiiiii</p><p class="short">iiiiiiii</p>`))
	assert.True(t, fonts.HasWebFont("test web mono"))
	assert.False(t, fonts.HasWebFont("Test Web Cyrillic"), "only fonts with Latin letters are loaded")
	bboxes := map[string]image.Rectangle{}
	families := map[string]rich.Family{}
	img := b.RenderImage(func() {
		b.WalkDown(func(n tree.Node) bool {
			if tx, ok := n.(*core.Text); ok {
				cl := tx.Parent.AsTree().Property("class")
				if s, ok := cl.(string); ok {
					bboxes[s] = tx.Geom.ContentBBox
					families[s] = tx.Styles.Font.Family
				}
			}
			return tree.Continue
		})
	})
	// widths are the widths of the ink of the texts
	widths := map[string]int{}
	for cl, bb := range bboxes {
		for y := bb.Min.Y; y < bb.Max.Y; y++ {
			for x := bb.Min.X; x < bb.Max.X; x++ {
				if r, _, _, _ := img.At(x, y).RGBA(); r < 0x8000 {
					widths[cl] = max(widths[cl], x-bb.Min.X)
				}
			}
		}
	}
	assert.Equal(t, rich.Custom, families["web"])
	assert.Equal(t, rich.Custom, families["short"], "the font shorthand")
	assert.Equal(t, rich.Serif, families["plain"])
	assert.Greater(t, widths["web"], widths["plain"]*3/2, "monospace i are wider")
}
