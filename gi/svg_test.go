// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"

	"goki.dev/mat32/v2"
	"goki.dev/svg"
)

func TestSVG(t *testing.T) {
	sc := NewScene()
	sv := NewSVG(sc)
	sv.SVG.Root.ViewBox.Size.SetScalar(10)
	svg.NewCircle(&sv.SVG.Root).SetPos(mat32.Vec2{5, 5}).SetRadius(5)
	sc.AssertPixelsOnShow(t, filepath.Join("svg", "basic_circle"))
}
