// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"net/url"
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
