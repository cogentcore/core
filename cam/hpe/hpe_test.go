// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hpe

import (
	"testing"

	"cogentcore.org/core/xgo/tolassert"
)

func TestHPE(t *testing.T) {
	l, m, s := XYZToLMS(0.4, 0.6, 0.23)
	tolassert.Equal(t, float32(0.55117565), l)
	tolassert.Equal(t, float32(0.6287903), m)
	tolassert.Equal(t, float32(0.23), s)

	l, m, s = SRGBLinToLMS(0.63, 0.34, 0.91)
	tolassert.Equal(t, float32(0.44553143), l)
	tolassert.Equal(t, float32(0.4412107), m)
	tolassert.Equal(t, float32(0.917642), s)

	l, m, s = SRGBToLMS(0.12, 0.41, 0.86)
	tolassert.Equal(t, float32(0.12346371), l)
	tolassert.Equal(t, float32(0.17244643), m)
	tolassert.Equal(t, float32(0.6923387), s)
}
