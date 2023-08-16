// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// adapted from: https://github.com/material-foundation/material-color-utilities
// Copyright 2022 Google LLC
// Licensed under the Apache License, Version 2.0 (the "License")

package cam16

// CAM represents a point in the cam16 color model along 6 dimensions
// representing the perceived hue, colorfulness, and brightness,
// similar to HSL but much more well-calibrated to actual human subjective judgments.
type CAM struct {

	// hue (h) is the spectral identity of the color (red, green, blue etc)
	Hue float32 `desc:"hue (h) is the spectral identity of the color (red, green, blue etc)"`

	// chroma (C) is the colorfulness or saturation of the color -- greyscale colors have no chroma, and fully saturated ones have high chroma
	Chroma float32 `desc:"chroma (C) is the colorfulness or saturation of the color -- greyscale colors have no chroma, and fully saturated ones have high chroma"`

	// colorfulness (M) is the absolute chromatic intensity
	Colorfulness float32 `desc:"colorfulness (M) is the absolute chromatic intensity"`

	// saturation (s) is the colorfulness relative to brightness
	Saturation float32 `desc:"saturation (s) is the colorfulness relative to brightness"`

	// brightness (Q) is the apparent amount of light from the color, which is not a simple function of actual light energy emitted
	Brightness float32 `desc:"brightness (Q) is the apparent amount of light from the color, which is not a simple function of actual light energy emitted"`

	// lightness (J) is the brightness relative to a reference white, which varies as a function of chroma and hue
	Lightness float32 `desc:"lightness (J) is the brightness relative to a reference white, which varies as a function of chroma and hue"`
}
