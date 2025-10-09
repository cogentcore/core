// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate core generate

// package pagesizes provides an enum of standard page sizes
// including image (e.g., 1080p, 4K, etc) and printed page sizes
// (e.g., A4, USLetter).
package pagesizes

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
)

// Sizes are standard physical drawing sizes
type Sizes int32 //enums:enum

const (
	// Custom =  nonstandard
	Custom Sizes = iota

	// Image 1280x720 Px = 720p
	Img1280x720

	// Image 1920x1080 Px = 1080p HD
	Img1920x1080

	// Image 3840x2160 Px = 4K
	Img3840x2160

	// Image 7680x4320 Px = 8K
	Img7680x4320

	// Image 1024x768 Px = XGA
	Img1024x768

	// Image 720x480 Px = DVD
	Img720x480

	// Image 640x480 Px = VGA
	Img640x480

	// Image 320x240 Px = old CRT
	Img320x240

	// A4 = 210 x 297 mm
	A4

	// USLetter = 8.5 x 11 in = 612 x 792 pt
	USLetter

	// USLegal = 8.5 x 14 in = 612 x 1008 pt
	USLegal

	// A0 = 841 x 1189 mm
	A0

	// A1 = 594 x 841 mm
	A1

	// A2 = 420 x 594 mm
	A2

	// A3 = 297 x 420 mm
	A3

	// A5 = 148 x 210 mm
	A5

	// A6 = 105 x 148 mm
	A6

	// A7 = 74 x 105
	A7

	// A8 = 52 x 74 mm
	A8

	// A9 = 37 x 52
	A9

	// A10 = 26 x 37
	A10
)

// Size returns the corresponding size values and units.
func (s Sizes) Size() (un units.Units, size math32.Vector2) {
	v := sizesMap[s]
	return v.un, math32.Vec2(v.x, v.y)
}

// Match returns a matching standard size for given units and dimension.
func Match(un units.Units, wd, ht float32) Sizes {
	trgl := values{un: un, x: wd, y: ht}
	trgp := values{un: un, x: ht, y: wd}
	for k, v := range sizesMap {
		if *v == trgl || *v == trgp {
			return k
		}
	}
	return Custom
}

// values are values for standard sizes
type values struct {
	un units.Units
	x  float32
	y  float32
}

// sizesMap is the map of size values for each standard size
var sizesMap = map[Sizes]*values{
	Img1280x720:  {units.UnitPx, 1280, 720},
	Img1920x1080: {units.UnitPx, 1920, 1080},
	Img3840x2160: {units.UnitPx, 3840, 2160},
	Img7680x4320: {units.UnitPx, 7680, 4320},
	Img1024x768:  {units.UnitPx, 1024, 768},
	Img720x480:   {units.UnitPx, 720, 480},
	Img640x480:   {units.UnitPx, 640, 480},
	Img320x240:   {units.UnitPx, 320, 240},
	A4:           {units.UnitMm, 210, 297},
	USLetter:     {units.UnitPt, 612, 792},
	USLegal:      {units.UnitPt, 612, 1008},
	A0:           {units.UnitMm, 841, 1189},
	A1:           {units.UnitMm, 594, 841},
	A2:           {units.UnitMm, 420, 594},
	A3:           {units.UnitMm, 297, 420},
	A5:           {units.UnitMm, 148, 210},
	A6:           {units.UnitMm, 105, 148},
	A7:           {units.UnitMm, 74, 105},
	A8:           {units.UnitMm, 52, 74},
	A9:           {units.UnitMm, 37, 52},
	A10:          {units.UnitMm, 26, 37},
}
