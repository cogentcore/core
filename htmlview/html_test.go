// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlview

import (
	"testing"

	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
)

func TestHTML(t *testing.T) {
	tests := map[string]string{
		"h1":     `<h1>Test</h1>`,
		"h2":     `<h2>Test</h2>`,
		"h3":     `<h3>Test</h3>`,
		"h4":     `<h4>Test</h4>`,
		"h5":     `<h5>Test</h5>`,
		"h6":     `<h6>Test</h6>`,
		"p":      `<p>Test</p>`,
		"ol":     `<ol><li>Test</li></ol>`,
		"ul":     `<ul><li>Test</li></ul>`,
		"button": `<button>Test</button>`,
		"input":  `<input value="Test">`,
	}
	for nm, s := range tests {
		t.Run(nm, func(t *testing.T) {
			b := core.NewBody()
			assert.NoError(t, ReadHTMLString(NewContext(), b, s))
			b.AssertRender(t, "html/"+nm)
		})
	}
}
