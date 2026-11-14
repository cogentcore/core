// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrefetchURLs tests the URLs of the resources that reading a page
// gets, for getting them in parallel before it is read: style sheets
// (with the imports and web fonts of the inline ones), scripts, and images
// (the candidates of their srcsets that are read), absolute, once each.
func TestPrefetchURLs(t *testing.T) {
	src := `<html><head>
<link rel="stylesheet" href="/a.css"><link rel="stylesheet" href="print.css" media="print">
<link rel="preload" as="font" href="/f.woff2"><link rel="icon" href="/i.png">
<style>@import url("b.css"); @font-face { font-family: X; src: url(x.woff2) format("woff2"), url(x.woff) }</style>
<script src="/s.js"></script><script src="/old.js" nomodule></script><script>inline()</script>
</head><body><img src="/one.png"><img src="/one.png"><img srcset="/small.png 1x, /big.png 2x">
<img src="data:image/png;base64,AAAA"></body></html>`
	ctx := NewContext()
	ctx.PageURL = "https://example.com/dir/page"
	ctx.WebFonts = true
	ctx.Scripting = true
	doc, err := ParseHTML(ctx, strings.NewReader(src))
	require.NoError(t, err)
	assert.Equal(t, []string{
		"https://example.com/a.css", "https://example.com/f.woff2",
		"https://example.com/dir/b.css", "https://example.com/dir/x.woff2",
		"https://example.com/s.js",
		"https://example.com/one.png", "https://example.com/small.png",
	}, ctx.PrefetchURLs(doc))
	ctx.Scripting = false
	assert.NotContains(t, ctx.PrefetchURLs(doc), "https://example.com/s.js", "without scripting")
}

// TestStyleSheetURLs tests the URLs that a style sheet gets: its imports
// (also in @media rules), and the first web font source of each
// @font-face rule that is read (not one only for other scripts than Latin,
// nor of a format that is not read), relative to the style sheet.
func TestStyleSheetURLs(t *testing.T) {
	css := `@import "base.css" screen;
@media (min-width: 1px) { @import url(more.css); }
@font-face { font-family: A; src: local(A), url(fonts/a.eot) format("embedded-opentype"), url(fonts/a.woff2) format("woff2") }
@font-face { font-family: Cyr; src: url(cyr.woff2); unicode-range: U+0400-04FF }
.x { color: red }`
	assert.Equal(t, []string{"https://cdn.example.com/css/base.css", "https://cdn.example.com/css/more.css", "https://cdn.example.com/css/fonts/a.woff2"},
		StyleSheetURLs(css, "https://cdn.example.com/css/site.css", true))
	assert.Equal(t, []string{"https://cdn.example.com/css/base.css", "https://cdn.example.com/css/more.css"},
		StyleSheetURLs(css, "https://cdn.example.com/css/site.css", false), "without web fonts")
}
