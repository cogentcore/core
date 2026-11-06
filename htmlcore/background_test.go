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
