// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/gist/colors"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
)

// MainStyleFunc is the main, gloabal style function
// that is called on all widgets to style them.
// By default, it is [gimain.StyleFunc], so if you
// wish to change styles without overriding all of the
// default styles, you should call [gimain.StyleFunc]
// at the start of your StyleFunc. For reference on
// how you should structure your StyleFunc, you
// should look at https://goki.dev/docs/gi/styling.
// Also, you can base your code on the code contained in
// [gimain.StyleFunc].
var MainStyleFunc func(w *WidgetBase)

// DefaultStyler describes an element that
// describes its default styles through the
// DefaultStyle function.
type DefaultStyler interface {
	// DefaultStyle applies the default styles
	// to the element.
	DefaultStyle()
}

// StyleFunc is the default style function
// for package gi that handles the styling
// of all widgets in it.
func StyleFunc(w *WidgetBase) {
	cs := CurrentColorScheme()
	fmt.Printf("Styling\t%v\n", w.This())
	// if par, ok := w.Parent().Embed(KiT_WidgetBase).(*WidgetBase); ok {
	// 	// fmt.Println("got parent")
	// 	if par.Style.Font.Color.IsNil() {
	// 		w.Style.Font.Color.SetColor(cs.Font)
	// 	} else {
	// 		// fmt.Println("inhereting color", par.Style.Font.Color)
	// 		w.Style.Font.Color.SetColor(par.Style.Font.Color)
	// 	}
	// }
	if ds, ok := w.This().(DefaultStyler); ok {
		ds.DefaultStyle()
	}
	// w.Style.Font.BgColor.SetColor(cs.Background)
	switch w := w.This().(type) {
	case *Viewport2D:
		// fmt.Println("styling viewport")
		// w.Style.Font.Color.SetColor(cs.Font)
	case *Label:
		styleLabel(w, cs)
	case *Icon:
		// fmt.Println("styling icon")
		styleIcon(w, cs)
	// case *Button:
	// 	styleButton(w, cs)
	// case *Action:
	// 	styleAction(w, cs)
	case *TextField:
		styleTextField(w, cs)
	}
}

func styleAction(a *Action, cs ColorScheme) {
	styleDefaultButton(&a.ButtonBase, cs)
	a.Style.Border.Style.Set(gist.BorderNone)
	a.Style.Border.Width.Set()
	a.Style.Border.Radius.Set()
	a.Style.Text.Align = gist.AlignCenter
	a.Style.Layout.Padding.Set(units.Px(2 * Prefs.DensityMultiplier()))
	a.Style.Layout.Margin.Set(units.Px(2 * Prefs.DensityMultiplier()))
	a.Style.Layout.MinWidth.SetEx(0.5)
	a.Style.Layout.MinHeight.SetEx(0.5)
}

func styleIcon(i *Icon, cs ColorScheme) {
	i.Style.Layout.Width.SetEm(1.5)
	i.Style.Layout.Height.SetEm(1.5)
	i.Style.Font.BgColor.SetColor(gist.Transparent)
	i.Style.Font.Color.SetColor(colors.White)
}

func styleButton(b *Button, cs ColorScheme) {
	b.Style.Border.Radius.Set(units.Px(5))
	b.Style.Border.Style.Set(gist.BorderNone)
	b.Style.Layout.Padding.Set(units.Px(5 * Prefs.DensityMultiplier()))
	// fmt.Println(b.State)
	if b.Type == ButtonPrimary {
		c := cs.Primary
		switch b.State {
		case ButtonHover:
			c = cs.Primary.Darker(20)
		case ButtonDown:
			c = cs.Primary.Darker(30)
		}
		b.Style.Font.Color.SetColor(cs.Font.Highlight(100))
		b.Style.Font.BgColor.SetColor(c)

	} else if b.Type == ButtonSecondary {
		b.Style.Font.Color.SetColor(cs.Primary)
		b.Style.Border.Color.Set(cs.Primary)
		b.Style.Border.Style.Set(gist.BorderSolid)
		b.Style.Border.Width.Set(units.Px(1))

		cc := cs.Background
		switch b.State {
		case ButtonHover:
			cc = cc.Highlight(20)
		case ButtonDown:
			cc = cc.Highlight(30)
		}
		b.Style.Font.BgColor.SetColor(cc)
	} else {
		styleDefaultButton(&b.ButtonBase, cs)
	}
	if icon, ok := b.Parts.ChildByType(KiT_Icon, ki.NoEmbeds, 0).(*Icon); ok {
		if ok {
			icon.Style.Layout.Width.SetEm(1)
			icon.Style.Layout.Height.SetEm(1)
			icon.Style.Layout.Margin.Set()
			icon.Style.Layout.Padding.Set()
		}
	}
}

func styleLabel(l *Label, cs ColorScheme) {
	switch p := l.Parent().Parent().(type) {
	case *Button:
		l.Style.Font.Color.SetColor(p.Style.Font.Color)
	}
	switch l.Type {
	case LabelP:
		l.Style.Font.Size.SetRem(1)
	case LabelLabel:
		l.Style.Font.Size.SetRem(0.75)
	case LabelH1:
		l.Style.Font.Size.SetRem(2)
		l.Style.Font.Weight = gist.WeightBold
	case LabelH2:
		l.Style.Font.Size.SetRem(1.5)
		l.Style.Font.Weight = gist.WeightBold
	case LabelH3:
		l.Style.Font.Size.SetRem(1.25)
		l.Style.Font.Weight = gist.WeightBold
	}
}

func styleTextField(tf *TextField, cs ColorScheme) {
	tf.Style.Border.Width.Set(units.Px(1))
	tf.CursorWidth.SetPx(3)
	tf.Style.Border.Color.Set(cs.Font.Highlight(30))
	tf.Style.Layout.Width.SetEm(20)
	tf.Style.Layout.Height.SetPx(2)
	tf.Style.Layout.Padding.Set(units.Px(4))
	tf.Style.Layout.Margin.Set(units.Px(1))
	tf.Style.Text.Align = gist.AlignLeft
	tf.Style.Font.BgColor.SetColor(cs.Background.Highlight(5))

	// fmt.Println("text field kids", tf.Parts.Kids)

	clear, ok := tf.Parts.ChildByName("clear", 1).(*Action)
	if ok {
		clear.StyleFunc = func() {
			clear.Style.Layout.Width.SetEx(0.5)
			clear.Style.Layout.Height.SetEx(0.5)
			clear.Style.Layout.MinWidth.SetEx(0.5)
			clear.Style.Layout.MinHeight.SetEx(0.5)
			clear.Style.Layout.Margin.Set()
			clear.Style.Layout.Padding.Set()
			clear.Style.Layout.AlignV = gist.AlignMiddle
		}
		// fmt.Println("clear", clear)

	}

	// space, ok := tf.Parts.ChildByName("space", 2).(*)

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

// // StyleFunc is the global style function
// // that can be set to specify custom styles for
// // widgets based on their characteristics.
// // It is set by default to DefaultStyleFunc, so if you
// // wish to change styles without overriding all of the
// // default styles, you should call DefaultStyleFunc
// // at the start of your StyleFunc. For reference on
// // how you should structure your StyleFunc, you
// // should look at https://goki.dev/docs/gi/styling.
// // Also, you can base your code on the code contained in
// // DefaultStyleFunc.
// var StyleFunc func(w *WidgetBase) = DefaultStyleFunc
