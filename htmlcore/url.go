// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"io/fs"
	"net/http"
	"net/url"
	"strings"
)

// parseRelativeURL parses the given raw URL relative to the given base URL.
func parseRelativeURL(rawURL, base string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return u, err
	}
	b, err := url.Parse(base)
	if err != nil {
		return u, err
	}
	return b.ResolveReference(u), nil
}

// GetURLWithFS returns a function suitable for [Context.GetURL] that gets
// resources from the given file system.
func GetURLWithFS(fsys fs.FS) func(rawURL string) (*http.Response, error) {
	return func(rawURL string) (*http.Response, error) {
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, err
		}
		if u.Scheme != "" {
			return http.Get(rawURL)
		}
		rawURL = strings.TrimPrefix(rawURL, "/")
		f, err := fsys.Open(rawURL)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			Status:        http.StatusText(http.StatusOK),
			StatusCode:    http.StatusOK,
			Body:          f,
			ContentLength: -1,
		}, nil
	}
}
