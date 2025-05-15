// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package gradient

import (
	"syscall/js"

	"cogentcore.org/core/colors"
)

// ToJS converts the gradient to a JavaScript object.
func (l *Linear) ToJS(ctx js.Value) js.Value {
	grad := ctx.Call("createLinearGradient", l.rStart.X, l.rStart.Y, l.rEnd.X, l.rEnd.Y)
	for _, stop := range l.Stops {
		grad.Call("addColorStop", stop.Pos, colors.AsHex(stop.Color))
	}
	return grad
}
