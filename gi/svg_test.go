// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"

	"goki.dev/goki/mat32"
	"goki.dev/goki/svg"
)

func TestSVG(t *testing.T) {
	sc := NewScene()
	sv := NewSVG(sc)
	sv.SVG.Root.ViewBox.Size.SetScalar(10)
	svg.NewCircle(&sv.SVG.Root).SetPos(mat32.V2(5, 5)).SetRadius(5)
	sc.AssertPixelsOnShow(t, filepath.Join("svg", "basic_circle"))
}
