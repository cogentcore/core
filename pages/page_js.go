// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package pages

import (
	"net/url"
	"strings"
	"syscall/js"

	"cogentcore.org/core/base/errors"
)

// firstPage is the first [Page] used for [getWebURL] or [saveWebURL],
// which is used to prevent nested [Page] widgets from incorrectly affecting the URL.
var firstPage *Page

var documentData = js.Global().Get("document").Get("documentElement").Get("dataset")

func init() {
	getWebURL = func(p *Page) string {
		if firstPage == nil {
			firstPage = p
		}
		if firstPage != p {
			return "/"
		}
		full, base, err := getURL()
		if errors.Log(err) != nil {
			return "/"
		}
		result := strings.TrimPrefix(full.String(), base.String())
		return "/" + result
	}
	saveWebURL = func(p *Page, u string) {
		if firstPage == nil {
			firstPage = p
		}
		if firstPage != p {
			return
		}
		_, base, err := getURL()
		if errors.Log(err) != nil {
			return
		}
		new, err := url.Parse(u)
		if errors.Log(err) != nil {
			return
		}
		fullNew := base.ResolveReference(new)
		js.Global().Get("history").Call("pushState", "", "", fullNew.String())
	}
}

// originalBase is used to cache the first website base URL to prevent
// issues with invalidation of the base URL caused by the data-base-path
// attribute not updating when a new page is loaded (because it is a
// Single-Page Application (SPA) so it doesn't load anything new).
var originalBase *url.URL

// getURL returns the full current URL and website base URL.
func getURL() (full, base *url.URL, err error) {
	full, err = url.Parse(js.Global().Get("location").Get("href").String())
	if err != nil {
		return
	}
	if originalBase != nil {
		base = originalBase
		return
	}
	basePath, err := url.Parse(documentData.Get("basePath").String())
	if err != nil {
		return
	}
	base = full.ResolveReference(basePath)
	originalBase = base

	// We must apply our new absolute base path to all of the links so
	// that the favicon updates correctly on future page changes.
	documentData.Set("basePath", base.String())
	links := js.Global().Get("document").Get("head").Call("getElementsByTagName", "link")
	for i := range links.Length() {
		link := links.Index(i)
		// Get("href") returns the absolute version, so we can just Set it directly.
		link.Set("href", link.Get("href").String())
	}

	return
}
