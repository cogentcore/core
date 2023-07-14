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
	fmt.Println(w.Nm, ki.Type(w))
	w.Style.Font.Color.SetColor(cs.Font)
	w.Style.Font.BgColor.SetColor(cs.Background)
	switch w := w.This().(type) {
	case *Viewport2D:
		fmt.Println("styling viewport")
		// w.Style.Font.Color.SetColor(cs.Font)
	case *Label:
		switch p := w.Parent().Parent().(type) {
		case *Button:
			w.Style.Font.Color.SetColor(p.Style.Font.Color)
		}
	case *Button:
		w.Style.Border.Radius.Set(units.Px(5))
		w.Style.Border.Style.Set(gist.BorderNone)
		w.Style.Layout.Padding.Set(units.Px(5))
		fmt.Println(w.State)
		if w.HasClass("primary") {
			c := cs.Primary
			switch w.State {
			case ButtonHover:
				c = cs.Primary.Darker(20)
			case ButtonDown:
				c = cs.Primary.Darker(30)
			}
			w.Style.Font.Color.SetColor(c.ContrastColor())
			w.Style.Font.BgColor.SetColor(c)

		} else if w.HasClass("secondary") {
			w.Style.Font.Color.SetColor(cs.Primary)
			w.Style.Border.Color.Set(cs.Primary)
			w.Style.Border.Style.Set(gist.BorderSolid)
			w.Style.Border.Width.Set(units.Px(1))

			cc := cs.Primary.ContrastColor()
			switch w.State {
			case ButtonHover:
				cc = cc.Highlight(20)
			case ButtonDown:
				cc = cc.Highlight(30)
			}
			w.Style.Font.BgColor.SetColor(cc)
		} else {
			styleDefaultButton(&w.ButtonBase, cs)
		}
	case *Action:
		w.Style.Layout.Padding.Set(units.Px(5))
		styleDefaultButton(&w.ButtonBase, cs)
	}
}

func styleDefaultButton(bb *ButtonBase, cs ColorScheme) {
	bb.Style.Font.Color.SetColor(cs.Font)
	bc := cs.Background.Highlight(5)
	switch bb.State {
	case ButtonHover:
		bc = bc.Highlight(10)
	case ButtonDown:
		bc = bc.Highlight(20)
	}
	bb.Style.Font.BgColor.SetColor(bc)
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
