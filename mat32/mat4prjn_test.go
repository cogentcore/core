// Copyright 2021 The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mat32

import (
	"testing"
)

func TestMat4Prjn(t *testing.T) {
	pts := []Vec3{{0.0, 0.0, 0.0}, {1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {0.5, 0.5, 0.5}, {-0.5, -0.5, -0.5}, {1, 1, 1}, {-1, -1, -1}}

	campos := V3(0, 0, 10)
	target := V3(0, 0, 0)
	var lookq Quat
	lookq.SetFromRotationMatrix(NewLookAt(campos, target, V3(0, 1, 0)))
	scale := V3(1, 1, 1)
	var cview Mat4
	cview.SetTransform(campos, lookq, scale)
	view, _ := cview.Inverse()

	var glprjn Mat4
	glprjn.SetPerspective(90, 1.5, 0.01, 100)

	var proj Mat4
	proj.MulMatrices(&glprjn, view)

	for _, pt := range pts {
		pjpt := pt.MulMat4(&proj)
		_ = pjpt
		// fmt.Printf("pt: %v\t   pj: %v\n", pt, pjpt)
	}
}
