// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAncestorKeys tests the keys that the elements that the elements that
// a selector selects are in must have: of the compound selectors before
// combinators of descendants and children, not of siblings, nor in
// parentheses.
func TestAncestorKeys(t *testing.T) {
	keys := func(ks ...string) KeyMask {
		var m KeyMask
		for _, k := range ks {
			m = m.Or(KeyBit(k))
		}
		return m
	}
	assert.Equal(t, keys(".menu"), AncestorKeys(".menu .item"))
	assert.Equal(t, keys("div", "ul"), AncestorKeys("div > ul li.x"))
	assert.Equal(t, keys(".b"), AncestorKeys(".a + .b .c"))
	assert.Equal(t, keys(".a", "#main"), AncestorKeys(".a:hover #main>.b"))
	assert.Equal(t, KeyMask{}, AncestorKeys(":not(.x) .y"))
	assert.Equal(t, KeyMask{}, AncestorKeys(".a ~ .b"))
	assert.Equal(t, KeyMask{}, AncestorKeys(".item"))
	assert.Equal(t, KeyMask{}, AncestorKeys(`.a\:b .c`), "escapes")
}
