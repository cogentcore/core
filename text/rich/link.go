// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import "cogentcore.org/core/text/textpos"

// LinkRec represents a hyperlink within shaped text.
type LinkRec struct {
	// Label is the text label for the link.
	Label string

	// URL is the full URL for the link.
	URL string

	// Properties are additional properties defined for the link,
	// e.g., from the parsed HTML attributes.
	// Properties map[string]any

	// Range defines the starting and ending positions of the link,
	// in terms of source rune indexes.
	Range textpos.Range
}

// GetLinks gets all the links from the source.
func (tx Text) GetLinks() []LinkRec {
	var lks []LinkRec
	n := tx.NumSpans()
	for i := range n {
		s, _ := tx.Span(i)
		if s.Special != Link {
			continue
		}
		lk := LinkRec{}
		lk.URL = s.URL
		lk.Range.Start = i
		for j := i + 1; j < n; j++ {
			e, _ := tx.Span(i)
			if e.Special == End {
				lk.Range.End = j
				break
			}
		}
		if lk.Range.End == 0 { // shouldn't happen
			lk.Range.End = i + 1
		}
		lk.Label = string(tx[lk.Range.Start:lk.Range.End].Join())
		lks = append(lks, lk)
	}
	return lks
}
