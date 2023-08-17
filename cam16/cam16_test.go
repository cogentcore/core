// Copyright (c) 2021, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cam16

import (
	"testing"

	"github.com/goki/cam/cie"
	"github.com/goki/mat32"
)

func expect(t *testing.T, ref, val float32) {
	if mat32.Abs(ref-val) > 0.001 {
		t.Errorf("expected value: %g != %g\n", ref, val)
	}
}

func TestView(t *testing.T) {
	vw := NewStdView()
	// fmt.Printf("%#v\n", vw)
	expect(t, 11.725676537, vw.AdaptingLuminance)
	expect(t, 50.000000000, vw.BgLuminance)
	expect(t, 2.000000000, vw.Surround)
	expect(t, 0.184186503, vw.BgYToWhiteY)
	expect(t, 29.981000900, vw.AW)
	expect(t, 1.016919255, vw.NBB)
	expect(t, 1.016919255, vw.NCB)
	expect(t, 0.689999998, vw.C)
	expect(t, 1.000000000, vw.NC)
	expect(t, 0.388481468, vw.FL)
	expect(t, 0.789482653, vw.FLRoot)
	expect(t, 1.909169555, vw.Z)

	expect(t, 1.021177769, vw.RGBD.X)
	expect(t, 0.986307740, vw.RGBD.Y)
	expect(t, 0.933960497, vw.RGBD.Z)
}

func TestCAM(t *testing.T) {
	// vw := NewStdView()
	camw := XYZToCAM(cie.SRGB100ToXYZ(1, 1, 1))
	// fmt.Printf("%#v\n", camw)
	expect(t, 100, camw.Lightness)
	expect(t, 2.869, camw.Chroma)
	expect(t, 209.492, camw.Hue)
	expect(t, 2.265, camw.Colorfulness)
	expect(t, 12.068, camw.Saturation)
	expect(t, 155.521, camw.Brightness)
}
