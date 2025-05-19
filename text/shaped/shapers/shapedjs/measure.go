// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"syscall/js"
)

// https://developer.mozilla.org/en-US/docs/Web/API/TextMetrics

// Metrics are html canvas MeasureText metrics.
type Metrics struct {
	// Width is the actual width of the text span, which appears to be equivalent to the
	// Advance in go-text.
	Width float32

	// ActualBoundingBoxLeft is the distance parallel to the baseline from the
	// alignment point given by the CanvasRenderingContext2D.textAlign property
	// to the left side of the bounding rectangle of the given text, in CSS pixels.
	// Positive numbers indicating a distance going left from the given alignment point.
	ActualBoundingBoxLeft float32

	// ActualBoundingBoxRight is the distance from the alignment point given by the
	// CanvasRenderingContext2D.textAlign property to the right side of the bounding
	// rectangle of the given text, in CSS pixels.
	// The distance is measured parallel to the baseline.
	ActualBoundingBoxRight float32

	// FontBoundingBoxAscent is the distance from the horizontal line indicated
	// by the CanvasRenderingContext2D.textBaseline attribute to the top of the
	// highest bounding rectangle of all the fonts used to render the text, in CSS pixels.
	FontBoundingBoxAscent float32

	// FontBoundingBoxDescent is the distance from the horizontal line indicated
	// by the CanvasRenderingContext2D.textBaseline attribute to the bottom of the
	// bounding rectangle of all the fonts used to render the text, in CSS pixels.
	FontBoundingBoxDescent float32

	// ActualBoundingBoxAscent is the distance from the horizontal line indicated
	// by the CanvasRenderingContext2D.textBaseline attribute to the top of the
	// highest bounding rectangle of the actual text, in CSS pixels.
	ActualBoundingBoxAscent float32

	// ActualBoundingBoxDescent is the distance from the horizontal line indicated
	// by the CanvasRenderingContext2D.textBaseline attribute to the bottom of the
	// bounding rectangle of all the fonts used to render the text, in CSS pixels.
	ActualBoundingBoxDescent float32
}

// MeasureText calls html canvas MeasureText functon on given canvas context,
// using currently set font.
func MeasureText(ctx js.Value, txt string) *Metrics {
	jm := ctx.Call("measureText", txt)

	m := &Metrics{
		Width:                    float32(jm.Get("width").Float()),
		ActualBoundingBoxLeft:    float32(jm.Get("actualBoundingBoxLeft").Float()),
		ActualBoundingBoxRight:   float32(jm.Get("actualBoundingBoxRight").Float()),
		FontBoundingBoxAscent:    float32(jm.Get("fontBoundingBoxAscent").Float()),
		FontBoundingBoxDescent:   float32(jm.Get("fontBoundingBoxDescent").Float()),
		ActualBoundingBoxAscent:  float32(jm.Get("actualBoundingBoxAscent").Float()),
		ActualBoundingBoxDescent: float32(jm.Get("actualBoundingBoxDescent").Float()),
	}
	// note: these are not widely supported.
	// EmHeightAscent:           float32(jm.Get("emHeightAscent").Float()),
	// EmHeightDescent:     float32(jm.Get("emHeightDescent").Float()),
	return m
}
