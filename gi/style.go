// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"

	"github.com/goki/ki/ki"
)

// DefaultStyleFunc is the default style function
// that is called to style widgets. When setting a custom
// style function in StyleFunc, you need to call this function
// to keep the default styles and build on top of them.
// If you wish to completely remove the default styles, you
// should not call this function in StyleFunc.
func DefaultStyleFunc(w *WidgetBase) {
	cs := CurrentColorScheme()
	w.Style.Font.Color.SetColor(cs.Font)
	w.Style.Font.BgColor.SetColor(cs.Background)
	fmt.Println(w.Nm, ki.Type(w))
	switch w := w.This().(type) {
	case *Button:
		w.Style.Border.Radius.Set(units.Px(5))
		w.Style.Border.Style.Set(gist.BorderNone)
		w.Style.Layout.Padding.Set(units.Px(5))
		if w.HasClass("primary") {
			w.Style.Font.Color.SetColor(cs.Primary.FontColor())
			w.Style.Font.BgColor.SetColor(cs.Primary)
		} else if w.HasClass("secondary") {
			w.Style.Font.Color.SetColor(cs.Primary)
			w.Style.Font.BgColor.SetColor(cs.Primary.FontColor())
			w.Style.Border.Style.Set(gist.BorderSolid)
			w.Style.Border.Color.Set(cs.Primary)
			w.Style.Border.Width.Set(units.Px(1))
		} else {
			w.Style.Font.Color.SetColor(cs.Font)
			w.Style.Font.BgColor.SetColor(cs.Background.Highlight(10))
		}

	}
}

// StyleFunc is the global style function
// that can be set to specify custom styles for
// widgets based on their characteristics.
// It is set by default to DefaultStyleFunc, so if you
// wish to change styles without overriding all of the
// default styles, you should call DefaultStyleFunc
// at the start of your StyleFunc. For reference on
// how you should structure your StyleFunc, you
// should look at https://goki.dev/docs/gi/styling.
// Also, you can base your code on the code contained in
// DefaultStyleFunc.
var StyleFunc func(w *WidgetBase) = DefaultStyleFunc
