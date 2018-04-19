// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"

	"github.com/rcoreilly/goki/gi"
	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/oswin/driver"
)

func main() {
	driver.Main(func(s oswin.Screen) {
		width = 800
		height = 800
		win := gi.NewWindow2D("test window", width, height)
		if err != nil {
			log.Fatal(err)
		}
		defer win.OSWin.Release()

		win.UpdateStart()
		vp := win.WinViewport2D()
		vp.Fill = true

		vlay := vpfill.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
		vlay.Lay = gi.LayoutCol

		rlay := vlay.AddNewChild(gi.KiT_Layout, "rowlay").(*gi.Layout)
		rlay.Lay = gi.LayoutRow
		rlay.SetProp("text-align", "center")
		edit1 := rlay.AddNewChild(gi.KiT_TextField, "edit1").(*gi.TextField)
		button1 := rlay.AddNewChild(gi.KiT_Button, "button1").(*gi.Button)
		button2 := rlay.AddNewChild(gi.KiT_Button, "button2").(*gi.Button)
		slider1 := rlay.AddNewChild(gi.KiT_Slider, "slider1").(*gi.Slider)

		edit1.Text = "Edit this text"
		edit1.SetProp("min-width", "20em")
		button1.Text = "Button 1"
		button2.Text = "Button 2"
		slider1.Dim = gi.X
		slider1.SetProp("width", "20em")
		slider1.SetValue(0.5)

		win.UpdateEnd()

		win.StartEventLoop()
	})
}
