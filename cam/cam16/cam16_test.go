// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cam16

import (
	"testing"

	"cogentcore.org/core/cam/cie"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/xgo/tolassert"
	"github.com/stretchr/testify/assert"
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

	nvw := *vw
	nvw.Surround = 0.5
	nvw.Update()
	expect(t, 0.55749995, nvw.C)
}

func TestCAM(t *testing.T) {
	camw := FromSRGB(1, 1, 1)
	expect(t, 209.492, camw.Hue)
	expect(t, 2.869, camw.Chroma)
	expect(t, 100, camw.Lightness)
	expect(t, 2.265, camw.Colorfulness)
	expect(t, 12.068, camw.Saturation)
	expect(t, 155.521, camw.Brightness)

	camr := FromSRGB(1, 0, 0)
	expect(t, 27.408, camr.Hue)
	expect(t, 113.354, camr.Chroma)
	expect(t, 46.445, camr.Lightness)
	expect(t, 89.490, camr.Colorfulness)
	expect(t, 91.889, camr.Saturation)
	expect(t, 105.988, camr.Brightness)

	camg := FromSRGB(0, 1, 0)
	expect(t, 142.139, camg.Hue)
	expect(t, 108.406, camg.Chroma)
	expect(t, 79.331, camg.Lightness)
	expect(t, 85.584, camg.Colorfulness)
	expect(t, 78.604, camg.Saturation)
	expect(t, 138.520, camg.Brightness)

	camb := FromSRGB(0, 0, 1)
	expect(t, 282.788, camb.Hue)
	expect(t, 87.227, camb.Chroma)
	expect(t, 25.465, camb.Lightness)
	expect(t, 68.864, camb.Colorfulness)
	expect(t, 93.674, camb.Saturation)
	expect(t, 78.481, camb.Brightness)

	r, g, b, a := (&CAM{40, 60, 80, 50, 45, 55}).RGBA()
	assert.Equal(t, uint32(0xef5a), r)
	assert.Equal(t, uint32(0x7439), g)
	assert.Equal(t, uint32(0x3f53), b)
	assert.Equal(t, uint32(0xffff), a)

	assert.Equal(t, FromJCHView(60, 50, 40, NewStdView()), FromJCH(60, 50, 40))
}

func TestXYZ(t *testing.T) {
	tests := [][3]float32{{0.5, 0.1, 0.6}, {0.3, 0.5, 0.1}, {0.4, 0.2, 0.8}, {0.777, 0.424, 0.521}}
	for _, test := range tests {
		x, y, z := cie.SRGBToXYZ(test[0], test[1], test[2])
		cam := FromXYZ(x, y, z)
		xc, yc, zc := cam.XYZ()
		expect(t, x, xc)
		expect(t, y, yc)
		expect(t, z, zc)
		rf, gf, bf := cie.XYZToSRGB(x, y, z)
		expect(t, rf, test[0])
		expect(t, gf, test[1])
		expect(t, bf, test[2])
		// r, g, b, a := cie.SRGBFloatToUint8(rf, gf, bf, 1)
		// fmt.Println(rf, gf, bf, r, g, b, a)
	}
}

func TestUCS(t *testing.T) {
	tests := [][3]float32{{1, 1, 0}, {0, 0, 1}} // , {0.4, 0.2, 0.8}, {0.777, 0.424, 0.521}}
	for _, test := range tests {
		x, y, z := cie.SRGBToXYZ(test[0], test[1], test[2])
		cam := FromXYZ(x, y, z)
		j, _, a, b := cam.UCS()
		ccam := FromUCS(j, a, b)
		// fmt.Printf("srgb: %g, %g, %g, cam: %#v  ccam: %#v\n", test[0], test[1], test[2], *cam, *ccam)
		expect(t, cam.Chroma, ccam.Chroma)
		expect(t, cam.Lightness, ccam.Lightness)
		expect(t, cam.Colorfulness, ccam.Colorfulness)
		expect(t, cam.Saturation, ccam.Saturation)
		expect(t, cam.Brightness, ccam.Brightness)
	}
}

func TestLMS(t *testing.T) {
	x, y, z := LMSToXYZ(0.25, 0.68, 0.47)
	tolassert.Equal(t, float32(-0.15201962), x)
	tolassert.Equal(t, float32(0.5152482), y)
	tolassert.Equal(t, float32(0.4663193), z)

	tolassert.Equal(t, float32(28.158047), InverseChromaticAdapt(52.1))
}

func TestSanitize(t *testing.T) {
	tolassert.Equal(t, float32(80), SanitizeDegrees(800))
	tolassert.Equal(t, float32(3.141593), SanitizeRadians(5*mat32.Pi))
}

func TestInCyclicOrder(t *testing.T) {
	assert.True(t, InCyclicOrder(0, 1, 2))
	assert.False(t, InCyclicOrder(0, 2, 1))
}
