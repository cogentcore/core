// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package coredom

import (
	"bytes"

	"go.abhg.dev/goldmark/wikilink"
)

// ResolveWikilink implements [wikilink.Resolver].
func (c *Context) ResolveWikilink(w *wikilink.Node) (destination []byte, err error) {
	if c.WikilinkResolver != nil {
		return c.WikilinkResolver(w)
	}
	return
}

// PkgGoDevWikilink returns a wikilink resolver that points to the pkg.go.dev page
// for the given project URL (eg: cogentcore.org/core)
func PkgGoDevWikilink(url string) func(w *wikilink.Node) (destination []byte, err error) {
	return func(w *wikilink.Node) (destination []byte, err error) {
		// pkg.go.dev uses fragments for first dot within package
		t := bytes.Replace(w.Target, []byte{'.'}, []byte{'#'}, 1)
		return append([]byte("https://pkg.go.dev/"+url+"/"), t...), nil
	}
}
