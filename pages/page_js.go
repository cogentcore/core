// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package pages

import (
	"net/url"
	"path"
	"strings"
	"syscall/js"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/pages/wpath"
)

func init() {
	getWebURL = func() string {
		full, base, err := getURL()
		if errors.Log(err) != nil {
			return "/"
		}
		result := strings.TrimPrefix(full.String(), base.String())
		return "/" + result
	}
	saveWebURL = func(u string) {
		_, base, err := getURL()
		if errors.Log(err) != nil {
			return
		}
		new, err := url.Parse(u)
		if errors.Log(err) != nil {
			return
		}

		// We must first apply all our new base path to all of the links so
		// that the favicon updates correctly.
		newBasePath := wpath.BasePath(u)
		links := js.Global().Get("document").Get("head").Call("getElementsByTagName", "link")
		for i := range links.Length() {
			link := links.Index(i)
			href := link.Get("href").String()
			relative := strings.TrimPrefix(href, base.String())
			newHref := path.Join(newBasePath, relative)
			link.Set("href", newHref)
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
	basePath, err := url.Parse(js.Global().Get("document").Get("documentElement").Get("dataset").Get("basePath").String())
	if err != nil {
		return
	}
	base = full.ResolveReference(basePath)
	originalBase = base
	return
}
