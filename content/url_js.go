// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package content

import (
	"net/url"
	"strings"
	"syscall/js"

	"cogentcore.org/core/base/errors"
)

// firstContent is the first [Content] used for [Content.getWebURL] or [Content.saveWebURL],
// which is used to prevent nested [Content] widgets from incorrectly affecting the URL.
var firstContent *Content

var documentData = js.Global().Get("document").Get("documentElement").Get("dataset")

// getWebURL returns the current relative web URL that should be passed to [Content.Open]
// on startup and in [Content.handleWebPopState].
func (ct *Content) getWebURL() string {
	if firstContent == nil {
		firstContent = ct
	}
	if firstContent != ct {
		return ""
	}
	full, base, err := getURL()
	if errors.Log(err) != nil {
		return ""
	}
	return strings.TrimPrefix(full.String(), base.String())
}

// saveWebURL saves the current page URL to the user's address bar and history.
func (ct *Content) saveWebURL() {
	if firstContent == nil {
		firstContent = ct
	}
	if firstContent != ct {
		return
	}
	url := ct.currentPage.URL
	if url == "" {
		url = ".."
	}
	js.Global().Get("history").Call("pushState", "", "", url)
}

// handleWebPopState adds a JS event listener to handle user navigation in the browser.
func (ct *Content) handleWebPopState() {
	js.Global().Get("window").Call("addEventListener", "popstate", js.FuncOf(func(this js.Value, args []js.Value) any {
		ct.open(ct.getWebURL(), false) // do not add to history while navigating history
		return nil
	}))
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
		// Get returns the absolute version, so we can just call Set with it
		// to update the href to actually be the absolute version.
		link.Set("href", link.Get("href").String())
	}
	return
}
