// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

// CustomConfigStyles is the custom, global style configuration function
// that is called on all widgets to configure their style functions.
// By default, it is nil. If you set it, you should mostly call
// AddStyleFunc within it. For reference on
// how you should structure your CustomStyleFunc, you
// should look at https://goki.dev/docs/gi/styling.
var CustomConfigStyles func(w *WidgetBase)

// // DefaultStyler describes an element that
// // describes its default styles through the
// // DefaultStyle function.
// type DefaultStyler interface {
// 	// DefaultStyle applies the default styles
// 	// to the element.
// 	DefaultStyle()
// }

// // StyleFunc is the default style function
// // for package gi that handles the styling
// // of all widgets in it.
// func StyleFunc(w *WidgetBase) {
// 	cs := CurrentColorScheme()
// 	// fmt.Printf("Styling\t%v\n", w.This())
// 	if par, ok := w.Parent().Embed(KiT_WidgetBase).(*WidgetBase); ok {
// 		// fmt.Println("got parent")
// 		if par.Style.Font.Color.IsNil() {
// 			w.Style.Font.Color.SetColor(cs.Font)
// 		} else {
// 			// fmt.Println("inhereting color", par.Style.Font.Color)
// 			w.Style.Font.Color.SetColor(par.Style.Font.Color)
// 		}
// 		w.Style.Text.WhiteSpace = par.Style.Text.WhiteSpace
// 	}
// 	if ds, ok := w.This().(DefaultStyler); ok {
// 		ds.DefaultStyle()
// 	}
// 	// w.Style.Font.BgColor.SetColor(cs.Background)
// 	switch w := w.This().(type) {
// 	case *Viewport2D:
// 		// fmt.Println("styling viewport")
// 		// w.Style.Font.Color.SetColor(cs.Font)
// 	case *Label:
// 		styleLabel(w, cs)
// 	case *Icon:
// 		// fmt.Println("styling icon")
// 		styleIcon(w, cs)
// 	// case *Button:
// 	// 	styleButton(w, cs)
// 	// case *Action:
// 	// 	styleAction(w, cs)
// 	case *TextField:
// 		styleTextField(w, cs)
// 	}
// }

// func styleAction(a *Action, cs ColorScheme) {
// 	styleDefaultButton(&a.ButtonBase, cs)
// 	a.Style.Border.Style.Set(gist.BorderNone)
// 	a.Style.Border.Width.Set()
// 	a.Style.Border.Radius.Set()
// 	a.Style.Text.Align = gist.AlignCenter
// 	a.Style.Padding.Set(units.Px(2 * Prefs.DensityMultiplier()))
// 	a.Style.Margin.Set(units.Px(2 * Prefs.DensityMultiplier()))
// 	a.Style.MinWidth.SetEx(0.5)
// 	a.Style.MinHeight.SetEx(0.5)
// }

// func styleIcon(i *Icon, cs ColorScheme) {
// 	i.Style.Width.SetEm(1.5)
// 	i.Style.Height.SetEm(1.5)
// 	i.Style.Font.BgColor.SetColor(gist.Transparent)
// 	i.Style.Font.Color.SetColor(colors.White)
// }

// func styleButton(b *Button, cs ColorScheme) {
// 	b.Style.Border.Radius.Set(units.Px(5))
// 	b.Style.Border.Style.Set(gist.BorderNone)
// 	b.Style.Padding.Set(units.Px(5 * Prefs.DensityMultiplier()))
// 	// fmt.Println(b.State)
// 	if b.Type == ButtonPrimary {
// 		c := cs.Primary
// 		switch b.State {
// 		case ButtonHover:
// 			c = cs.Primary.Darker(20)
// 		case ButtonDown:
// 			c = cs.Primary.Darker(30)
// 		}
// 		b.Style.Font.Color.SetColor(cs.Font.Highlight(100))
// 		b.Style.Font.BgColor.SetColor(c)

// 	} else if b.Type == ButtonSecondary {
// 		b.Style.Font.Color.SetColor(cs.Primary)
// 		b.Style.Border.Color.Set(cs.Primary)
// 		b.Style.Border.Style.Set(gist.BorderSolid)
// 		b.Style.Border.Width.Set(units.Px(1))

// 		cc := cs.Background
// 		switch b.State {
// 		case ButtonHover:
// 			cc = cc.Highlight(20)
// 		case ButtonDown:
// 			cc = cc.Highlight(30)
// 		}
// 		b.Style.Font.BgColor.SetColor(cc)
// 	} else {
// 		styleDefaultButton(&b.ButtonBase, cs)
// 	}
// 	if icon, ok := b.Parts.ChildByType(KiT_Icon, ki.NoEmbeds, 0).(*Icon); ok {
// 		if ok {
// 			icon.Style.Width.SetEm(1)
// 			icon.Style.Height.SetEm(1)
// 			icon.Style.Margin.Set()
// 			icon.Style.Padding.Set()
// 		}
// 	}
// }

// func styleLabel(l *Label, cs ColorScheme) {
// 	switch p := l.Parent().Parent().(type) {
// 	case *Button:
// 		l.Style.Font.Color.SetColor(p.Style.Font.Color)
// 	}
// 	switch l.Type {
// 	case LabelP:
// 		l.Style.Font.Size.SetRem(1)
// 	case LabelLabel:
// 		l.Style.Font.Size.SetRem(0.75)
// 	case LabelH1:
// 		l.Style.Font.Size.SetRem(2)
// 		l.Style.Font.Weight = gist.WeightBold
// 	case LabelH2:
// 		l.Style.Font.Size.SetRem(1.5)
// 		l.Style.Font.Weight = gist.WeightBold
// 	case LabelH3:
// 		l.Style.Font.Size.SetRem(1.25)
// 		l.Style.Font.Weight = gist.WeightBold
// 	}
// }

// func styleTextField(tf *TextField, cs ColorScheme) {
// 	tf.Style.Border.Width.Set(units.Px(1))
// 	tf.CursorWidth.SetPx(3)
// 	tf.Style.Border.Color.Set(cs.Font.Highlight(30))
// 	tf.Style.Width.SetEm(20)
// 	tf.Style.Height.SetPx(2)
// 	tf.Style.Padding.Set(units.Px(4))
// 	tf.Style.Margin.Set(units.Px(1))
// 	tf.Style.Text.Align = gist.AlignLeft
// 	tf.Style.Font.BgColor.SetColor(cs.Background.Highlight(5))

// 	// fmt.Println("text field kids", tf.Parts.Kids)

// 	clear, ok := tf.Parts.ChildByName("clear", 1).(*Action)
// 	if ok {
// 		clear.StyleFunc = func() {
// 			clear.Style.Width.SetEx(0.5)
// 			clear.Style.Height.SetEx(0.5)
// 			clear.Style.MinWidth.SetEx(0.5)
// 			clear.Style.MinHeight.SetEx(0.5)
// 			clear.Style.Margin.Set()
// 			clear.Style.Padding.Set()
// 			clear.Style.AlignV = gist.AlignMiddle
// 		}
// 		// fmt.Println("clear", clear)

// 	}

// 	// space, ok := tf.Parts.ChildByName("space", 2).(*)

// }

// func styleDefaultButton(bb *ButtonBase, cs ColorScheme) {
// 	bb.Style.Font.Color.SetColor(cs.Font)
// 	bc := cs.Background.Highlight(5)
// 	switch bb.State {
// 	case ButtonHover:
// 		bc = bc.Highlight(10)
// 	case ButtonDown:
// 		bc = bc.Highlight(20)
// 	}
// 	bb.Style.Font.BgColor.SetColor(bc)
// }

// // // StyleFunc is the global style function
// // // that can be set to specify custom styles for
// // // widgets based on their characteristics.
// // // It is set by default to DefaultStyleFunc, so if you
// // // wish to change styles without overriding all of the
// // // default styles, you should call DefaultStyleFunc
// // // at the start of your StyleFunc. For reference on
// // // how you should structure your StyleFunc, you
// // // should look at https://goki.dev/docs/gi/styling.
// // // Also, you can base your code on the code contained in
// // // DefaultStyleFunc.
// // var StyleFunc func(w *WidgetBase) = DefaultStyleFunc
