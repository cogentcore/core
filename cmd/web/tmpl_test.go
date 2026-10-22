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
