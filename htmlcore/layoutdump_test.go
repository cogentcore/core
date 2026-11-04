// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// TestLayoutDump prints the layout geometry of the widgets for the HTML in
// the LAYOUT_HTML environment variable, read with [Context.BrowserStyle],
// for debugging layout:
//
//	LAYOUT_HTML='<p>Some <b>text</b></p>' go test -run TestLayoutDump -v
func TestLayoutDump(t *testing.T) {
	src := os.Getenv("LAYOUT_HTML")
	if f := os.Getenv("LAYOUT_FILE"); f != "" {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src = string(b)
	}
	if src == "" {
		t.Skip()
	}
	b := core.NewBody()
	b.Scene.Styler(func(s *styles.Style) { s.Padding.Zero() })
	b.Styler(func(s *styles.Style) { s.Padding.Zero(); s.Gap.Zero(); s.Min.X.Dp(800); s.Max.X.Dp(800) })
	ctx := NewContext()
	ctx.BrowserStyle = true
	ctx.MediaWidth = 800
	if f := os.Getenv("LAYOUT_FILE"); f != "" {
		if u, err := os.ReadFile(strings.TrimSuffix(f, ".html") + ".url"); err == nil {
			ctx.PageURL = strings.TrimSpace(string(u))
		}
	}
	ReadHTMLString(ctx, b, src)
	img := b.RenderImage(func() {
		fmt.Println("scene", b.Scene.Geom.Size.Actual.Total)
		b.WalkDown(func(n tree.Node) bool {
			w, ok := n.(core.Widget)
			if !ok {
				return tree.Continue
			}
			wb := w.AsWidget()
			depth := 0
			for p := wb.Parent; p != nil; p = p.AsTree().Parent {
				depth++
			}
			s := &wb.Styles
			if tx, ok := w.(*core.Text); ok {
				txt := tx.Text
				if len(txt) > 30 {
					txt = txt[:30]
				}
				fmt.Printf("%*s%q\n", depth*2, "", txt)
			}
			fmt.Printf("%*s%s %v %v pos %v act %v alloc %v grow %v gw %v min %v max %v marg %v pad %v disp %v dir %v ovf %v op %v color %v\n", depth*2, "", wb.Name, wb.Property("tag"), wb.Property("class"),
				wb.Geom.Pos.Total, wb.Geom.Size.Actual.Total, wb.Geom.Size.Alloc.Total, s.Grow, s.GrowWrap, s.Min.Dots(), s.Max.Dots(), s.Margin.Dots(), s.Padding.Dots(), s.Display, s.Direction, s.Overflow, s.Opacity, s.Color)
			return tree.Continue
		})
	})
	fmt.Println("img", img.Bounds())
}
