// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cie

import (
	"testing"

	"cogentcore.org/core/xgo/tolassert"
)

func TestLAB(t *testing.T) {
	tolassert.Equal(t, float32(0.887904), LABCompress(0.7))
	tolassert.Equal(t, float32(0.1379544), LABCompress(0.000003))
	tolassert.Equal(t, float32(0.21600002), LABUncompress(0.6))

	l, a, b := XYZToLAB(0.1, 0.3, 0.5)
	tolassert.Equal(t, float32(61.65422), l)
	tolassert.Equal(t, float32(-98.673805), a)
	tolassert.Equal(t, float32(-20.413673), b)

	x, y, z := LABToXYZ(28, 14, 36.2)
	tolassert.Equal(t, float32(0.06422656), x)
	tolassert.Equal(t, float32(0.054573778), y)
	tolassert.Equal(t, float32(0.008442593), z)

	tolassert.Equal(t, float32(2.3023312), LToY(17))
	tolassert.Equal(t, float32(21.579498), YToL(3.4))
}
