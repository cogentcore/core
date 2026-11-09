// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSupportsMatches(t *testing.T) {
	for cond, want := range map[string]bool{
		"(display: grid)":                               true,
		"not (display:grid)":                            false,
		"(display: grid) and (not (display: flex))":     false,
		"not ((display: grid) or (x: y))":               false,
		"(-moz-appearance: none)":                       false,
		"not (-moz-appearance: none)":                   true,
		"(-webkit-touch-callout: none)":                 false,
		"(position: -moz-sticky) or (position: sticky)": true,
		"selector(:has(a))":                             true,
		"not selector(:has(a))":                         false,
		"base":                                          true,
		"":                                              true,
	} {
		assert.Equal(t, want, supportsMatches(cond), cond)
	}
}

func TestResolveGridLines(t *testing.T) {
	names := gridLineNumbers("[viewport-start] minmax(0px, 1fr) [content-start left-column-start] repeat(2, 60px) [left-column-end main-column-start] repeat(12, 60px) [content-end main-column-end] minmax(0px, 1fr) [viewport-end]", nil)
	assert.Equal(t, []int{2}, names["content-start"])
	assert.Equal(t, []int{16}, names["main-column-end"])
	assert.Equal(t, "2 / 16", resolveGridLines("content-start / main-column-end", names))
	assert.Equal(t, "2 / 16", resolveGridLines("content", names))
	assert.Equal(t, "4 / 16", resolveGridLines("main-column", names))
	assert.Equal(t, "1 / span 2", resolveGridLines("1 / span 2", names))
	assert.Equal(t, "auto / 16", resolveGridLines("nothing / content-end", names))
	assert.Equal(t, "3", resolveGridLines("3", names))
}

func TestAcceptHeaderNoAVIF(t *testing.T) {
	// image servers send AVIF to those who accept it, which is not decoded
	assert.NotContains(t, acceptHeader, "avif")
	assert.Contains(t, acceptHeader, "image/webp")
}
