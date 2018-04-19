// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"reflect"

	"github.com/rcoreilly/goki/gi"
	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/oswin/driver"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/kit"
)

func main() {
	driver.Main(func(s oswin.Screen) {
		width := 800
		height := 800

		// turn this on to see a trace of the rendering
		// gi.Update2DTrace = true
		// gi.Render2DTrace = true
		// gi.Layout2DTrace = true

		rec := ki.Node{}          // receiver for events
		rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

		win := gi.NewWindow2D("GoGi Widgets Window", width, height)
		win.UpdateStart()

		icnm := "widget-wedge-down"
		wdicon := gi.IconByName(icnm)

		vp := win.WinViewport2D()
		vp.SetProp("background-color", "#FFF")
		vp.Fill = true

		vlay := vp.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
		vlay.Lay = gi.LayoutCol

		row1 := vlay.AddNewChild(gi.KiT_Layout, "row1").(*gi.Layout)
		row1.Lay = gi.LayoutRow
		row1.SetProp("align-vert", gi.AlignMiddle)
		row1.SetProp("align-horiz", gi.AlignCenter)
		row1.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
		row1.SetStretchMaxWidth()

		spc := vlay.AddNewChild(gi.KiT_Space, "spc1").(*gi.Space)
		spc.SetFixedHeight(units.NewValue(2.0, units.Em))

		row1.AddNewChild(gi.KiT_Stretch, "str1")
		lab1 := row1.AddNewChild(gi.KiT_Label, "lab1").(*gi.Label)
		lab1.Text = "This is a demonstration of the various GoGi Widgets"
		lab1.SetProp("max-width", -1)
		lab1.SetProp("text-align", "center")
		row1.AddNewChild(gi.KiT_Stretch, "str2")

		row2 := vlay.AddNewChild(gi.KiT_Layout, "row2").(*gi.Layout)
		row2.Lay = gi.LayoutRow
		row2.SetProp("align-vert", "center")
		row2.SetProp("align-horiz", gi.AlignJustify)
		row2.SetProp("margin", 2.0)
		row2.SetStretchMaxWidth()

		mb1 := row2.AddNewChild(gi.KiT_MenuButton, "menubutton1").(*gi.MenuButton)

		mb1.SetText("Menu Button")
		mb1.AddMenuText("Menu Item 1", rec.This, 1, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

		mb1.AddMenuText("Menu Item 2", rec.This, 2, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

		mb1.AddMenuText("Menu Item 3", rec.This, 3, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

		mb1.SetProp("align-vert", gi.AlignMiddle)

		button1 := row2.AddNewChild(gi.KiT_Button, "button1").(*gi.Button)
		button2 := row2.AddNewChild(gi.KiT_Button, "button2").(*gi.Button)
		checkbox := row2.AddNewChild(gi.KiT_CheckBox, "checkbox").(*gi.CheckBox)
		checkbox.Text = "Toggle"
		edit1 := row2.AddNewChild(gi.KiT_TextField, "edit1").(*gi.TextField)

		edit1.Text = "Edit this text"
		edit1.SetProp("min-width", "20em")
		edit1.SetProp("align-vert", gi.AlignMiddle)
		// button1.SetText("Button 1")
		button1.SetIcon(wdicon)
		button2.SetText("Open GoGiEditor")
		button1.SetProp("align-vert", gi.AlignMiddle)
		button2.SetProp("align-vert", gi.AlignMiddle)
		checkbox.SetProp("align-vert", gi.AlignMiddle)

		edit1.TextFieldSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("Received line edit signal: %v from edit: %v with data: %v\n", gi.TextFieldSignals(sig), send.Name(), data)
		})

		button1.ButtonSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("Received button signal: %v from button: %v\n", gi.ButtonSignals(sig), send.Name())
			if sig == int64(gi.ButtonClicked) {
				gi.PromptDialog(vp, "Button1 Dialog", "This is a dialog!  Various specific types of dialogs are available.", true, true, nil, nil)
			}
		})

		button2.ButtonSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("Received button signal: %v from button: %v\n", gi.ButtonSignals(sig), send.Name())
			if sig == int64(gi.ButtonClicked) {
				gi.GoGiEditorOf(vp)
			}
		})

		row3 := vlay.AddNewChild(gi.KiT_Layout, "row3").(*gi.Layout)
		row3.Lay = gi.LayoutRow
		row3.SetProp("align-vert", "center")
		row3.SetProp("align-horiz", "left")
		row3.SetProp("margin", 2.0)
		row3.SetStretchMaxWidth()
		// row3.SetStretchMaxHeight()

		slider1 := row3.AddNewChild(gi.KiT_Slider, "slider1").(*gi.Slider)
		slider1.Dim = gi.X
		slider1.SetMinPrefWidth(units.NewValue(20, units.Em))
		slider1.Defaults()
		slider1.SetValue(0.5)
		slider1.Snap = true
		slider1.Tracking = true
		slider1.Icon = gi.IconByName("widget-circlebutton-on")

		slider2 := row3.AddNewChild(gi.KiT_Slider, "slider2").(*gi.Slider)
		slider2.Dim = gi.Y
		slider2.SetMinPrefHeight(units.NewValue(10, units.Em))
		slider2.SetStretchMaxHeight()
		slider2.Defaults()
		slider2.SetValue(0.5)

		slider1.SliderSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("Received slider signal: %v from slider: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
		})

		slider2.SliderSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("Received slider signal: %v from slider: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
		})

		scrollbar1 := row3.AddNewChild(gi.KiT_ScrollBar, "scrollbar1").(*gi.ScrollBar)
		scrollbar1.Dim = gi.X
		scrollbar1.SetMinPrefWidth(units.NewValue(20, units.Em))
		scrollbar1.SetFixedHeight(units.NewValue(20, units.Px))
		scrollbar1.Defaults()
		scrollbar1.SetThumbValue(0.25)
		scrollbar1.SetValue(0.25)
		// scrollbar1.Snap = true
		scrollbar1.Tracking = true

		scrollbar2 := row3.AddNewChild(gi.KiT_ScrollBar, "scrollbar2").(*gi.ScrollBar)
		scrollbar2.Dim = gi.Y
		scrollbar2.SetMinPrefHeight(units.NewValue(10, units.Em))
		scrollbar2.SetStretchMaxHeight()
		scrollbar2.Defaults()
		scrollbar2.SetThumbValue(0.1)
		scrollbar2.SetValue(0.5)

		scrollbar1.SliderSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("Received scrollbar signal: %v from scrollbar: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
		})

		scrollbar2.SliderSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("Received scrollbar signal: %v from scrollbar: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
		})

		row4 := vlay.AddNewChild(gi.KiT_Layout, "row4").(*gi.Layout)
		row4.Lay = gi.LayoutRow
		row4.SetProp("align-vert", "center")
		row4.SetProp("align-horiz", "left")
		row4.SetProp("margin", 2.0)
		row4.SetStretchMaxWidth()
		// row4.SetStretchMaxHeight()

		sb := row4.AddNewChild(gi.KiT_SpinBox, "spinbox").(*gi.SpinBox)
		sb.HasMin = true
		sb.Min = 0.0
		sb.SpinBoxSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("SpinBox %v value changed: %v\n", send.Name(), data)
		})

		cb := row4.AddNewChild(gi.KiT_ComboBox, "combobox").(*gi.ComboBox)
		cb.ItemsFromTypes(kit.Types.AllImplementersOf(reflect.TypeOf((*gi.Node2D)(nil)).Elem(), false), true, true, 50)
		cb.ComboSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("ComboBox %v selected index: %v data: %v\n", send.Name(), sig, data)
		})

		win.UpdateEnd()

		win.StartEventLoop()
	})
}
