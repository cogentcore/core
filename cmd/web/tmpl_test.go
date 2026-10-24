// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"strings"
	"testing"

	"cogentcore.org/core/cmd/config"
	"cogentcore.org/core/content/bcontent"
	"github.com/stretchr/testify/assert"
)

func TestIndexHTMLFontText(t *testing.T) {
	const meta = `<meta name="cogentcore-font-text"`
	c := &config.Config{}
	b, err := makeIndexHTML(c, "/", &bcontent.PreRenderPage{}, "")
	assert.NoError(t, err)
	assert.NotContains(t, string(b), meta)

	c.Web.FontText = `āΩ"`
	b, err = makeIndexHTML(c, "/", &bcontent.PreRenderPage{}, "")
	assert.NoError(t, err)
	assert.Contains(t, string(b), meta+` content="āΩ&#34;">`)
	assert.Equal(t, 1, strings.Count(string(b), meta))
}

func TestExternalResources(t *testing.T) {
	fonts := "https://fonts.googleapis.com/css2?family=Noto+Sans:ital,wght@0,400..700;1,400..700&family=Roboto+Mono:ital,wght@0,400..700;1,400..700&display=swap"
	assert.Equal(t, []string{fonts}, externalResources(indexHTML))
	assert.Equal(t, []string{
		fonts,
		"https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/katex.min.css",
		"https://cdn.jsdelivr.net/npm/vue@3.5.13/dist/vue.global.prod.js",
		"https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/katex.min.js",
		"https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/contrib/auto-render.min.js",
	}, externalResources(contentHTML))
}

func TestAppWorkerJS(t *testing.T) {
	c := &config.Config{}
	c.Build.Output = t.TempDir()
	c.Build.Target = []config.Platform{{OS: "wasm"}}
	b, err := makeAppWorkerJS(c, false)
	assert.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `"app.wasm"`)
	assert.NotContains(t, s, `"content.css"`)
	assert.NotContains(t, s, "{{")
	assert.Contains(t, s, `const cacheOrigins = ["https://fonts.googleapis.com","https://fonts.gstatic.com"];`)

	c.Build.Target = []config.Platform{{OS: "js"}}
	c.Web.Styles = []string{"https://example.com/font.css"}
	b, err = makeAppWorkerJS(c, true)
	assert.NoError(t, err)
	s = string(b)
	assert.Contains(t, s, `"app.gs.js"`)
	assert.NotContains(t, s, `"app.wasm"`)
	for _, f := range []string{"content.css", "content.js", "nav.js"} {
		assert.Contains(t, s, `"`+f+`"`)
	}
	assert.Contains(t, s, `"https://cdn.jsdelivr.net/npm/vue@3.5.13/dist/vue.global.prod.js"`)
	assert.Contains(t, s, `"https://example.com/font.css"`)
	assert.Contains(t, s, `const cacheOrigins = ["https://cdn.jsdelivr.net","https://example.com","https://fonts.googleapis.com","https://fonts.gstatic.com"];`)
}
