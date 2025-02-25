// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"fmt"
	"syscall/js"
)

// https://developer.mozilla.org/en-US/docs/Web/API/TextMetrics

func MeasureTest() {
	ctx := js.Global().Get("document").Call("getElementById", "app").Call("getContext", "2d") // todo
	m := MeasureText(ctx, "This is a test")
	fmt.Println(m)
}

type Metrics struct {
	Width float32
}

func MeasureText(ctx js.Value, txt string) *Metrics {
	jm := ctx.Call("measureText", txt)

	m := &Metrics{
		Width: float32(jm.Get("width").Float()),
	}
	return m
}
