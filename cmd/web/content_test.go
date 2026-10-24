// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasePath(t *testing.T) {
	assert.Equal(t, "", basePath(""))
	assert.Equal(t, "../", basePath("button"))
	assert.Equal(t, "../../", basePath("widgets/button"))
	assert.Equal(t, "./", pageHref("", ""))
	assert.Equal(t, "../", pageHref("../", ""))
	assert.Equal(t, "../widget/", pageHref("../", "widget"))
}

func TestRebaseURLs(t *testing.T) {
	tests := []struct{ in, out string }{
		{`<a href="widget" target="_blank">w</a>`, `<a href="../widget/">w</a>`},
		{`<a href="widget#types" target="_blank">w</a>`, `<a href="../widget/#types">w</a>`},
		{`<a href="button#types" target="_blank">t</a>`, `<a href="#types">t</a>`},
		{`<a href="" target="_blank">h</a>`, `<a href="../">h</a>`},
		{`<a id="x" href="https://example.com" target="_blank">e</a>`, `<a id="x" href="https://example.com" target="_blank">e</a>`},
		{`<a href="#figure_one">f</a>`, `<a href="#figure_one">f</a>`},
		{`<a href="mailto:a@b.c">m</a>`, `<a href="mailto:a@b.c">m</a>`},
		{`<img id="fig" src="media/fig.png" alt="x">`, `<img id="fig" src="../media/fig.png" alt="x">`},
		{`<p>href="widget"</p>`, `<p>href="widget"</p>`},
		{`<a href="media/fig.png" target="_blank">f</a>`, `<a href="../media/fig.png" target="_blank">f</a>`},
		{`<a href="docs/paper.v2.pdf#page=3" target="_blank">p</a>`, `<a href="../docs/paper.v2.pdf#page=3" target="_blank">p</a>`},
		{`<a href="v1.2" target="_blank">v</a>`, `<a href="../v1.2/">v</a>`}, // page URL with a dot
	}
	pages := map[string]bool{"": true, "button": true, "widget": true, "v1.2": true}
	for _, tt := range tests {
		assert.Equal(t, tt.out, rebaseURLs(tt.in, "../", "button", pages, nil), tt.in)
	}
	assert.Equal(t, `<a href="./">h</a>`, rebaseURLs(`<a href="">h</a>`, "", "widget", pages, nil))
	assert.Equal(t, `<a href="widget/">w</a>`, rebaseURLs(`<a href="widget">w</a>`, "", "", pages, nil))
}
