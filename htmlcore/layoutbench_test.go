// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
)

// BenchmarkLayoutDocsPage lays out long pages of the docs, as the window
// width changes (resize: the shaping of the texts is cached, but their
// wrappings at the new widths are not), and with the cache of shaped texts
// emptied before each layout (cold: as for the first layout of a page).
func BenchmarkLayoutDocsPage(b *testing.B) {
	widths := []float32{1024, 984, 944, 904, 864, 824}
	for _, page := range []string{"gpu", "principles", "keymaps"} {
		md, err := os.ReadFile(filepath.Join("..", "docs", "content", page+".md"))
		assert.NoError(b, err)
		// The front matter of the page is not part of its content.
		s := string(md)
		if strings.HasPrefix(s, "+++") {
			if i := strings.Index(s[3:], "+++"); i >= 0 {
				s = s[i+6:]
			}
		}
		for _, mode := range []string{"resize", "cold"} {
			b.Run(page+"/"+mode, func(b *testing.B) {
				body := core.NewBody()
				// Wikilinks link to themselves, as in the docs.
				ctx := NewContext()
				ctx.AddWikilinkHandler(func(text string) (string, string) {
					return text, text
				})
				assert.NoError(b, ReadMDString(ctx, body, s))
				body.AssertRender(b, "layout/"+page, func() {
					sc := body.Scene
					reset, _ := sc.TextShaper().(interface{ ResetCache() })
					size := math32.FromPoint(sc.SceneGeom.Size)
					body.AsyncLock()
					i := 0
					for b.Loop() {
						if mode == "cold" && reset != nil {
							reset.ResetCache()
						}
						sc.LayoutFrame(math32.Vec2(widths[i%len(widths)], size.Y))
						i++
					}
					// The page is rendered as laid out at the size of the window.
					sc.LayoutFrame(size)
					sc.NeedsRender()
					body.AsyncUnlock()
				})
			})
		}
	}
}
