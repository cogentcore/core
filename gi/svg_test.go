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
	sv.SVG = svg.NewSVG(500, 500)
	svg.NewCircle(&sv.SVG.Root).SetRadius(50).SetPos(mat32.Vec2{250, 250})
	sc.AssertPixelsOnShow(t, filepath.Join("svg", "basic_circle"))
}
