// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"fmt"
	"os"
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
	if src == "" {
		t.Skip()
	}
	b := core.NewBody()
	b.Scene.Styler(func(s *styles.Style) { s.Padding.Zero() })
	b.Styler(func(s *styles.Style) { s.Padding.Zero(); s.Gap.Zero(); s.Min.X.Dp(800); s.Max.X.Dp(800) })
	ctx := NewContext()
	ctx.BrowserStyle = true
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
			fmt.Printf("%*s%s %v pos %v act %v alloc %v grow %v gw %v min %v max %v marg %v pad %v\n", depth*2, "", wb.Name, wb.Property("tag"),
				wb.Geom.Pos.Total, wb.Geom.Size.Actual.Total, wb.Geom.Size.Alloc.Total, s.Grow, s.GrowWrap, s.Min.Dots(), s.Max.Dots(), s.Margin.Dots(), s.Padding.Dots())
			return tree.Continue
		})
	})
	fmt.Println("img", img.Bounds())
}
