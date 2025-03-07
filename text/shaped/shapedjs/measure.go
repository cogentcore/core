// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"fmt"
	"syscall/js"

	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// https://developer.mozilla.org/en-US/docs/Web/API/TextMetrics

// Metrics are html canvas MeasureText metrics.
type Metrics struct {
	Width                    float32
	ActualBoundingBoxLeft    float32
	ActualBoundingBoxRight   float32
	FontBoundingBoxAscent    float32
	FontBoundingboxDescent   float32
	ActualBoundingBoxAscent  float32
	ActualBoundingBoxDescent float32
	HangingBaseline          float32
	AlphabeticBaseline       float32
	IdeographicBaseline      float32

	// not widely supported:
	// EmHeightAscent           float32
	// EmHeightDescent          float32
}

func MeasureTest(txt string) {
	ctx := js.Global().Get("document").Call("getElementById", "app").Call("getContext", "2d") // todo
	fsty := rich.NewStyle()
	tsty := text.NewStyle()
	uc := units.Context{}
	uc.Defaults()
	tsty.ToDots(&uc)
	fn := NewFont(fsty, tsty, &rich.DefaultSettings)
	SetFontStyle(ctx, fn, tsty, 0)
	m := MeasureText(ctx, txt)
	fmt.Printf("%#v\n", m)
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
		FontBoundingboxDescent:   float32(jm.Get("fontBoundingBoxDescent").Float()),
		ActualBoundingBoxAscent:  float32(jm.Get("actualBoundingBoxAscent").Float()),
		ActualBoundingBoxDescent: float32(jm.Get("actualBoundingBoxDescent").Float()),
		HangingBaseline:          float32(jm.Get("hangingBaseline").Float()),
		AlphabeticBaseline:       float32(jm.Get("alphabeticBaseline").Float()),
		IdeographicBaseline:      float32(jm.Get("ideographicBaseline").Float()),
	}
	// note: these are not widely supported.
	// EmHeightAscent:           float32(jm.Get("emHeightAscent").Float()),
	// EmHeightDescent:     float32(jm.Get("emHeightDescent").Float()),
	return m
}
