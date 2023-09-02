// Copyright (c) 2021, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cam16

import (
	"testing"

	"goki.dev/mat32/v2"
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
	camw := SRGBToCAM(1, 1, 1)
	expect(t, 209.492, camw.Hue)
	expect(t, 2.869, camw.Chroma)
	expect(t, 100, camw.Lightness)
	expect(t, 2.265, camw.Colorfulness)
	expect(t, 12.068, camw.Saturation)
	expect(t, 155.521, camw.Brightness)

	camr := SRGBToCAM(1, 0, 0)
	expect(t, 27.408, camr.Hue)
	expect(t, 113.354, camr.Chroma)
	expect(t, 46.445, camr.Lightness)
	expect(t, 89.490, camr.Colorfulness)
	expect(t, 91.889, camr.Saturation)
	expect(t, 105.988, camr.Brightness)

	camg := SRGBToCAM(0, 1, 0)
	expect(t, 142.139, camg.Hue)
	expect(t, 108.406, camg.Chroma)
	expect(t, 79.331, camg.Lightness)
	expect(t, 85.584, camg.Colorfulness)
	expect(t, 78.604, camg.Saturation)
	expect(t, 138.520, camg.Brightness)

	camb := SRGBToCAM(0, 0, 1)
	expect(t, 282.788, camb.Hue)
	expect(t, 87.227, camb.Chroma)
	expect(t, 25.465, camb.Lightness)
	expect(t, 68.864, camb.Colorfulness)
	expect(t, 93.674, camb.Saturation)
	expect(t, 78.481, camb.Brightness)

}
