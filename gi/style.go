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

// StyleFunc is the default style function
// for package gi that handles the styling
// of all widgets in it.
func StyleFunc(w *WidgetBase) {
	cs := CurrentColorScheme()
	// fmt.Printf("%s %T %T\n", w.Nm, w.This(), w.Parent())
	w.Style.Font.Color.SetColor(cs.Font)
	w.Style.Font.BgColor.SetColor(cs.Background)
	switch w := w.This().(type) {
	case *Viewport2D:
		// fmt.Println("styling viewport")
		// w.Style.Font.Color.SetColor(cs.Font)
	case *Label:
		styleLabel(w, cs)
	case *Icon:
		// fmt.Println("styling icon")
		styleIcon(w, cs)
	case *Button:
		styleButton(w, cs)
	case *Action:
		w.Style.Layout.Padding.Set(units.Px(5))
		styleDefaultButton(&w.ButtonBase, cs)
		if _, ok := w.Parent().Parent().(*TextField); ok {
			fmt.Println("styling textfield icon")
			w.Style.Layout.Width.SetEx(0.5)
			w.Style.Layout.Height.SetEx(0.5)
			w.Style.Layout.Margin.Set()
			w.Style.Layout.Padding.Set()
			w.Style.Layout.AlignV = gist.AlignMiddle
		}
	case *TextField:
		styleTextField(w, cs)
	}
}

func styleIcon(i *Icon, cs ColorScheme) {
	i.Style.Layout.Width.SetEm(1.5)
	i.Style.Layout.Height.SetEm(1.5)
	i.Style.Font.BgColor.SetColor(gist.Transparent)
	i.Style.Font.Opacity = 0
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
	if iconk := b.Parts.ChildByType(KiT_Icon, ki.NoEmbeds, 0); iconk != nil {
		icon, ok := iconk.(*Icon)
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
	tf.Style.Border.Color.Set(cs.Border)
	tf.Style.Layout.Width.SetEm(20)
	tf.Style.Layout.Padding.Set(units.Px(4))
	tf.Style.Layout.Margin.Set(units.Px(1))
	tf.Style.Text.Align = gist.AlignLeft

	fmt.Println("text field kids", tf.Parts.Kids)

	// clear := tf.Parts.ChildByName("clear", 1).(*WidgetBase)
	// fmt.Println("clear", clear)
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
