// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tex_test

import (
	"testing"

	. "cogentcore.org/core/text/tex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tests in this file are fast (a few expressions, of about 0.3s each),
// so they always run, unlike the image tests of the many symbols, which need
// CORE_TEX_TESTS (see skipTex).

// TestLaTeXMathPaths tests that expressions are typeset as paths whose
// sizes relate as expected: wider for more symbols, taller for fractions
// and display style, and scaled by the font size.
func TestLaTeXMathPaths(t *testing.T) {
	width := func(expr string, size float32) (float32, float32) {
		t.Helper()
		p, err := LaTeXMath(expr, size)
		require.NoError(t, err, expr)
		require.False(t, p.Empty(), expr)
		b := p.FastBounds()
		return b.Size().X, b.Size().Y
	}
	xw, xh := width(`q`, 16)
	xxw, _ := width(`q+q+q`, 16)
	assert.Greater(t, xxw, 3*xw, "three symbols and two operators")
	_, fh := width(`\frac{q}{w}`, 16)
	assert.Greater(t, fh, 1.5*xh, "a fraction is taller")
	_, dh := width(`$\sum_{k=0}^{n} k$`, 16)
	_, ih := width(`\sum_{k=0}^{n} k`, 16)
	assert.Greater(t, dh, ih, "display style is taller than inline")
	bw, _ := width(`q`, 32)
	assert.InDelta(t, 2*xw, bw, float64(0.1*xw), "twice the font size")
}

// TestLaTeXMathErrors tests the error of an empty expression, and that
// the engine works after it.
func TestLaTeXMathErrors(t *testing.T) {
	_, err := LaTeXMath("  ", 16)
	assert.Error(t, err)
	p, err := LaTeXMath(`w^2_k`, 16)
	assert.NoError(t, err)
	assert.False(t, p.Empty())
}
