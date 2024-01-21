// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package coredom

import (
	"testing"

	"cogentcore.org/core/gi"
)

func TestRenderHTML(t *testing.T) {
	b := gi.NewBody("test-render-html")
	s := `
		<h1>Coredom</h1>
		<p>This is a demonstration of the various features of coredom</p>
		<button>Hello, world!</button>
		`
	err := ReadHTMLString(BaseContext(), b, s)
	if err != nil {
		t.Error(err)
	}
	b.AssertRender(t, "test-render-html")
}
