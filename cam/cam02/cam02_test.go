// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cam02

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXYZToLMS(t *testing.T) {
	x, y, z := XYZToLMS(0.3, 0.6, 0.5)
	assert.Equal(t, float32(0.39640003), x)
	assert.Equal(t, float32(0.8104701), y)
	assert.Equal(t, float32(0.50076), z)
}

func TestSRGBLinToLMS(t *testing.T) {
	l, m, s := SRGBLinToLMS(0.45, 0.2, 0.83)
	assert.Equal(t, float32(0.29307953), l)
	assert.Equal(t, float32(0.22564001), m)
	assert.Equal(t, float32(0.84963936), s)

}

func TestSRGBToLMS(t *testing.T) {
	l, m, s := SRGBToLMS(0.45, 0.2, 0.83)
	assert.Equal(t, float32(0.090681426), l)
	assert.Equal(t, float32(0.044864923), m)
	assert.Equal(t, float32(0.6354286), s)
}

func TestLuminanceAdapt(t *testing.T) {
	assert.Equal(t, float32(1), LuminanceAdapt(200))
}

func TestResponseCompression(t *testing.T) {
	assert.Equal(t, float32(0.23376063), ResponseCompression(0.86))
}

func TestLMSToResp(t *testing.T) {
	lc, mc, sc, lmc, grey := LMSToResp(0.1, 0.95, 0.56)
	assert.Equal(t, float32(0.39294684), lc)
	assert.Equal(t, float32(0.91159713), mc)
	assert.Equal(t, float32(0.14976773), sc)
	assert.Equal(t, float32(0.12970123), lmc)
	assert.Equal(t, float32(0.5916059), grey)
}

func TestSRGBToLMSResp(t *testing.T) {
	lc, mc, sc, lmc, grey := SRGBToLMSResp(0.23, 0.48, 0.88)
	assert.Equal(t, float32(0.44037464), lc)
	assert.Equal(t, float32(0.4746794), mc)
	assert.Equal(t, float32(0.1688544), sc)
	assert.Equal(t, float32(0.08960256), lmc)
	assert.Equal(t, float32(0.46925616), grey)
}
