// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package coredom

import (
	"net/url"
	"strings"
)

// IsURL returns whether the given [url.URL] is probably a URL
// (as opposed to just a normal string of text)
func IsURL(u *url.URL) bool {
	return u.Scheme != "" || u.Port() != "" || strings.Contains(u.String(), ".")
}

// NormalizeURL sets the scheme of the URL to "https" if it is unset.
func NormalizeURL(u *url.URL) {
	if u.Scheme == "" {
		u.Scheme = "https"
	}
}

// ParseURL parses and normalized the given raw URL. If the given RawURL
// is actually a URL (as specified by [IsURL]), it returns the URL and true.
// Otherwise, it returns nil, false.
func ParseURL(rawURL string) (*url.URL, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, false
	}
	if !IsURL(u) {
		return nil, false
	}
	NormalizeURL(u)
	return u, true
}

// ParseRelativeURL parses the given raw URL relative to the given base URL.
func ParseRelativeURL(rawURL, base string) (*url.URL, error) {
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
