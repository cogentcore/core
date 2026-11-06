// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"fmt"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

// TestBackgroundShorthand tests that the background shorthand without a
// color (as "0 0", with which github.com resets its buttons) makes the
// background transparent, and that one with a gradient (not supported)
// keeps the color before it.
func TestBackgroundShorthand(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	assert.NoError(t, ReadHTMLString(ctx, b, "<style>.reset{background:0 0}.grad{background:#336699;background:linear-gradient(red, blue)}</style>"+
		"<button class=\"reset\">reset</button><button class=\"plain\">plain</button><div class=\"grad\">grad</div>"))
	bg := map[string]string{}
	b.RenderImage(func() {
		b.WalkDown(func(n tree.Node) bool {
			if wb := core.AsWidget(n); wb != nil {
				if cl := fmt.Sprint(wb.Property("class")); cl == "reset" || cl == "plain" || cl == "grad" {
					c := "none"
					if wb.Styles.Background != nil {
						c = colors.AsHex(colors.ToUniform(wb.Styles.Background))
					}
					bg[cl] = c
				}
			}
			return tree.Continue
		})
	})
	assert.Contains(t, []string{"none", "#00000000"}, bg["reset"], "transparent")
	assert.Equal(t, "#EFEFEF", bg["plain"], "the default of buttons")
	assert.Equal(t, "#336699", bg["grad"], "the fallback color")
}

// TestComputedValue tests the values of properties as cascaded for the
// computed styles of scripts: of the rule that applies, inherited for
// inherited properties, and not for others.
func TestComputedValue(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	assert.NoError(t, ReadHTMLString(ctx, b, `<style>:root{--c:green} .a{text-align:center;color:var(--c);width:10px} .a .b{color:inherit;width:inherit} .c{text-align:initial}</style>`+
		`<div class="a"><h1 class="b">x</h1><p class="c" style="color: red !important">y</p></div>`))
	var doc *html.Node
	b.WalkDown(func(n tree.Node) bool {
		if nd, ok := core.AsWidget(n).Property("node").(*html.Node); ok && doc == nil {
			for doc = nd; doc.Parent != nil; doc = doc.Parent {
			}
		}
		return tree.Continue
	})
	h1 := selectNodes(doc, "h1")[0]
	p := selectNodes(doc, "p")[0]
	assert.Equal(t, "center", ctx.ComputedValue(h1, "text-align"), "inherited")
	assert.Equal(t, "green", ctx.ComputedValue(h1, "color"), "inherit, with var()")
	assert.Equal(t, "10px", ctx.ComputedValue(h1, "width"), "width: inherit")
	assert.Equal(t, "", ctx.ComputedValue(p, "width"), "not inherited")
	assert.Equal(t, "", ctx.ComputedValue(p, "text-align"), "initial")
	assert.Equal(t, "red", ctx.ComputedValue(p, "color"), "the inline style")
	assert.Equal(t, "green", ctx.ComputedValue(p, "--c"), "a custom property")
	assert.Equal(t, "", ctx.ComputedValue(p, "--C"), "custom properties are case sensitive")
}
