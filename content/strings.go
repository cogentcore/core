// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import "cogentcore.org/core/content/bcontent"

// hamming computes the Hamming distance between the two given strings.
// It is based on https://github.com/adrg/strutil/blob/master/metrics/hamming.go.
func hamming(a, b string) int {
	runesA, runesB := []rune(a), []rune(b)

	// Check if both terms are empty.
	lenA, lenB := len(runesA), len(runesB)
	if lenA == 0 && lenB == 0 {
		return 0
	}

	// If the lengths of the sequences are not equal, the distance is
	// initialized to their absolute difference. Otherwise, it is set to 0.
	if lenA > lenB {
		lenA, lenB = lenB, lenA
	}
	distance := lenB - lenA

	// Calculate Hamming distance.
	for i := 0; i < lenA; i++ {
		if runesA[i] != runesB[i] {
			distance++
		}
	}

	return distance
}

// similarPage returns the page most similar to the given URL, used for automatic 404 redirects.
func (ct *Content) similarPage(url string) *bcontent.Page {
	var best *bcontent.Page
	bestDistance := 1000000
	for _, page := range ct.pages {
		distance := hamming(url, page.URL)
		if distance < bestDistance {
			best = page
			bestDistance = distance
		}
	}
	return best
}
