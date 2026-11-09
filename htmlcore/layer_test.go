// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLayerRules tests that the rules in @layer, @container, @scope, and
// @starting-style blocks apply, including those nested in each other (as
// in the style sheet of github.com, whose @container rules in its @layer
// made the parser fail on the whole style sheet).
func TestLayerRules(t *testing.T) {
	ss := parseCSS("@layer base, brand;" +
		"@layer brand{.a{color:red}@container (width>=1280px){.b{color:blue}}.c{color:green}}" +
		"@scope (.x){.d{color:red}}@starting-style{.e{opacity:0}}.f{color:red}")
	var sels []string
	for _, r := range ss.Rules {
		sels = append(sels, r.Name+r.Prelude)
		for _, k := range r.Rules {
			sels = append(sels, "  "+k.Name+k.Prelude)
			for _, k2 := range k.Rules {
				sels = append(sels, "    "+k2.Prelude)
			}
		}
	}
	assert.Equal(t, []string{"@layerbase, brand", "@supportsbrand", "  .a", "  @supports--container: (width>=1280px)", "    .b", "  .c",
		"@supports(.x)", "  .d", "@supports", "  .e", ".f"}, sels)
}
