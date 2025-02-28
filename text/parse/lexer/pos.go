// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"cogentcore.org/core/text/token"
)

// EosPos is a line of EOS token positions, always sorted low-to-high
type EosPos []int

// FindGt returns any pos value greater than given token pos, -1 if none
func (ep EosPos) FindGt(ch int) int {
	for i := range ep {
		if ep[i] > ch {
			return ep[i]
		}
	}
	return -1
}

// FindGtEq returns any pos value greater than or equal to given token pos, -1 if none
func (ep EosPos) FindGtEq(ch int) int {
	for i := range ep {
		if ep[i] >= ch {
			return ep[i]
		}
	}
	return -1
}

////////  TokenMap

// TokenMap is a token map, for optimizing token exclusion
type TokenMap map[token.Tokens]struct{}

// Set sets map for given token
func (tm TokenMap) Set(tok token.Tokens) {
	tm[tok] = struct{}{}
}

// Has returns true if given token is in the map
func (tm TokenMap) Has(tok token.Tokens) bool {
	_, has := tm[tok]
	return has
}
