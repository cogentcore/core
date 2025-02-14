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
	n := len(tx)
	for si := range n {
		sp := RuneToSpecial(tx[si][0])
		if sp != Link {
			continue
		}
		lr := tx.SpecialRange(si)
		ls := tx[lr.Start:lr.End]
		s, _ := tx.Span(si)
		lk := LinkRec{}
		lk.URL = s.URL
		sr, _ := tx.Range(lr.Start)
		_, er := tx.Range(lr.End)
		lk.Range = textpos.Range{sr, er}
		lk.Label = string(ls.Join())
		lks = append(lks, lk)
		si = lr.End
	}
	return lks
}
