// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package coredom

import (
	"path/filepath"
	"testing"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/ki"
)

func TestRenderHTML(t *testing.T) {
	b := gi.NewBody("test-render-html")
	s := `
		<h1>Coredom</h1>
		<p>This is a demonstration of the various features of coredom</p>
		<button>Hello, world!</button>
		`
	grr.Test(t, ReadHTMLString(NewContext(), b, s))
	b.AssertRender(t, "test-render-html")
}

func TestHTMLElements(t *testing.T) {
	tests := []string{
		`<h1>Test</h1>`,
		`<h2>Test</h2>`,
		`<h3>Test</h3>`,
	}
	for _, s := range tests {
		b := gi.NewBody()
		grr.Test(t, ReadHTMLString(NewContext(), b, s))

		// find the tag of the element
		tag := ""
		b.WalkPre(func(k ki.Ki) bool {
			if tp := k.Prop("tag"); tp != nil && tp != "body" {
				tag = tp.(string)
				return ki.Break
			}
			return ki.Continue
		})

		b.AssertRender(t, filepath.Join("html", "elements", tag))
	}
}
