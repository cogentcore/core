// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texcache

import (
	"slices"
	"testing"

	"cogentcore.org/core/paint/ppath"
	"github.com/stretchr/testify/assert"
)

// TestGetSizes checks that the paths of an expression in different sizes are
// the same in any order of use, and are scaled from the path as added.
func TestGetSizes(t *testing.T) {
	p, err := ppath.ParseSVGPath("M0.3 1.7L10.1 2.9L5.3 13.7z")
	assert.NoError(t, err)
	Add("x^2", 16, p)
	defer delete(cache, "x^2")

	want28 := p.Clone().Scale(28.0/16, 28.0/16)
	for range 3 {
		assert.True(t, slices.Equal(p, Get("x^2", 16)))
		assert.True(t, slices.Equal(want28, Get("x^2", 28)))
	}

	// The returned paths are copies, which the caller can change.
	Get("x^2", 28).Translate(5, 5)
	Get("x^2", 16).Translate(5, 5)
	assert.True(t, slices.Equal(want28, Get("x^2", 28)))
	assert.True(t, slices.Equal(p, Get("x^2", 16)))
	assert.Nil(t, Get("y^2", 16))
}
