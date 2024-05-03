// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gradient

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
)

func BenchmarkLinear(b *testing.B) {
	g := NewLinear().AddStop(colors.White, 0).AddStop(colors.Black, 1)
	g.Update(1, math32.B2(0, 0, 100, 100), math32.Identity2())
	for range b.N {
		g.At(40, 40)
	}
}
