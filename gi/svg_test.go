// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"

	"cogentcore.org/core/mat32"
	"cogentcore.org/core/svg"
)

func TestSVG(t *testing.T) {
	b := NewBody()
	sv := NewSVG(b)
	sv.SVG.Root.ViewBox.Size.SetScalar(10)
	svg.NewCircle(&sv.SVG.Root).SetPos(mat32.V2(5, 5)).SetRadius(5)
	b.AssertRender(t, filepath.Join("svg", "basic_circle"))
}
