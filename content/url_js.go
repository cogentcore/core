// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package content

import (
	"fmt"
	"net/url"
	"syscall/js"
)

var documentData = js.Global().Get("document").Get("documentElement").Get("dataset")

// saveWebURL saves the current page URL to the user's address bar and history.
func (ct *Content) saveWebURL() {
	url := ct.currentPage.URL
	if url == "" {
		url = ".."
	}
	js.Global().Get("history").Call("pushState", "", "", url)
}

// handleWebPopState adds a JS event listener to handle user navigation in the browser.
func (ct *Content) handleWebPopState() {
	js.Global().Get("window").Call("addEventListener", "popstate", js.FuncOf(func(this js.Value, args []js.Value) any {
		full, base, err := getURL()
		fmt.Println(full, base, err)
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
