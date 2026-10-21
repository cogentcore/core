// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package imagex

import (
	"bytes"
	"image"
	"image/color"
	"syscall/js"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeBrowser(t *testing.T) {
	if js.Global().Get("createImageBitmap").IsUndefined() {
		t.Skip("createImageBitmap is not available")
	}
	// The tolerance is per channel: the browser and Go may decode lossy
	// formats slightly differently, as JPEG chroma upsampling, so up to
	// 1% of their samples may also differ by more.
	formats := map[Formats]int{PNG: 0, GIF: 0, BMP: 0, JPEG: 8}
	for format, tol := range formats {
		var buf bytes.Buffer
		assert.NoError(t, Write(testImage(), &buf, format))
		b := buf.Bytes()
		file := format.String()
		im, _, err := Read(bytes.NewReader(b))
		assert.NoError(t, err)
		jim, ok := im.(*JSRGBA)
		if !assert.True(t, ok, "%s is decoded by the browser", file) {
			continue
		}
		assert.Nil(t, jim.RGBA, "%s pixels are read back only when needed", file)

		goim, _, err := image.Decode(bytes.NewReader(b))
		assert.NoError(t, err)
		assert.Equal(t, goim.Bounds().Size(), im.Bounds().Size(), file)

		// The pixels read back from the bitmap match Go's decoding.
		bad, n := 0, 0
		sz := goim.Bounds().Size()
		for y := 0; y < sz.Y; y += 7 {
			for x := 0; x < sz.X; x += 7 {
				n++
				a := color.RGBAModel.Convert(goim.At(goim.Bounds().Min.X+x, goim.Bounds().Min.Y+y)).(color.RGBA)
				c := color.RGBAModel.Convert(im.At(x, y)).(color.RGBA)
				if diff(a.R, c.R) > tol || diff(a.G, c.G) > tol || diff(a.B, c.B) > tol || diff(a.A, c.A) > tol {
					bad++
				}
			}
		}
		if tol == 0 {
			assert.Zero(t, bad, "%s pixels differ", file)
		} else {
			assert.LessOrEqual(t, bad, max(1, n/100), "%s pixels differ", file)
		}
	}
}

func diff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
