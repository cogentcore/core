// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"cogentcore.org/core/content/bcontent"
	"github.com/adrg/strutil/metrics"
)

// similarPage returns the page most similar to the given URL, used for automatic 404 redirects.
func (ct *Content) similarPage(url string) *bcontent.Page {
	m := metrics.NewJaccard()
	m.CaseSensitive = false

	var best *bcontent.Page
	bestSimilarity := -1.0
	for _, page := range ct.pages {
		similarity := m.Compare(url, page.URL)
		if similarity > bestSimilarity {
			best = page
			bestSimilarity = similarity
		}
	}
	return best
}
