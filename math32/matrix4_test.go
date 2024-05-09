// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import "testing"

func TestMatrix4Projection(t *testing.T) {
	pts := []Vector3{{0.0, 0.0, 0.0}, {1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {0.5, 0.5, 0.5}, {-0.5, -0.5, -0.5}, {1, 1, 1}, {-1, -1, -1}}

	campos := Vec3(0, 0, 10)
	target := Vec3(0, 0, 0)
	var lookq Quat
	lookq.SetFromRotationMatrix(NewLookAt(campos, target, Vec3(0, 1, 0)))
	scale := Vec3(1, 1, 1)
	var cview Matrix4
	cview.SetTransform(campos, lookq, scale)
	view, _ := cview.Inverse()

	var glprojection Matrix4
	glprojection.SetPerspective(90, 1.5, 0.01, 100)

	var proj Matrix4
	proj.MulMatrices(&glprojection, view)

	for _, pt := range pts {
		pjpt := pt.MulMatrix4(&proj)
		_ = pjpt
		// fmt.Printf("pt: %v\t   pj: %v\n", pt, pjpt)
	}
}
