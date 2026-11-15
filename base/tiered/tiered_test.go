// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tiered

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTiered(t *testing.T) {
	tr := Tiered[[]string]{First: []string{"f"}, Normal: []string{"n"}, Final: []string{"l"}}
	var order []string
	tr.Do(func(s *[]string) {
		order = append(order, *s...)
		*s = append(*s, "x")
	})
	assert.Equal(t, []string{"f", "n", "l"}, order, "first, normal, then final")
	assert.Equal(t, []string{"n", "x"}, tr.Normal, "the tiers are changed in place")

	other := Tiered[[]string]{First: []string{"F"}, Normal: []string{"N"}, Final: []string{"L"}}
	tr.DoWith(&other, func(a, b *[]string) {
		*a = append(*a, *b...)
	})
	assert.Equal(t, []string{"f", "x", "F"}, tr.First)
	assert.Equal(t, []string{"l", "x", "L"}, tr.Final)
}
