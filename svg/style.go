// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"image/color"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist/colors"
)

// StyleFunc is the default style function
// for package svg that handles the styling
// of all widgets in it.
func StyleFunc(w *gi.WidgetBase) {
	// cs := gi.CurrentColorScheme()
	switch w := w.This().(type) {
	case *Icon:
		w.Style.BackgroundColor.SetColor(color.Transparent)
		w.Style.Color.SetColor(colors.White)
	}
}
