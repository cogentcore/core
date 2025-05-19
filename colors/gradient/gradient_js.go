// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package gradient

import (
	"syscall/js"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
)

// ToJS converts the gradient to a JavaScript object.
func (l *Linear) ToJS(ctx js.Value) js.Value {
	grad := ctx.Call("createLinearGradient", l.rStart.X, l.rStart.Y, l.rEnd.X, l.rEnd.Y)
	for _, stop := range l.Stops {
		if math32.IsNaN(stop.Pos) {
			continue
		}
		grad.Call("addColorStop", stop.Pos, colors.AsHex(stop.Color))
	}
	return grad
}

// ToJS converts the gradient to a JavaScript object.
func (r *Radial) ToJS(ctx js.Value) js.Value {
	grad := ctx.Call("createRadialGradient", r.rCenter.X, r.rCenter.Y, 0, r.rFocal.X, r.rFocal.Y, r.rRadius.X) // TODO: is this the right use of center and focal?
	for _, stop := range r.Stops {
		if math32.IsNaN(stop.Pos) {
			continue
		}
		grad.Call("addColorStop", stop.Pos, colors.AsHex(stop.Color))
	}
	return grad
}
