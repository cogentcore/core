// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate goki generate

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/mat32/v2"
)

func main() { gimain.Run(app) }

// TableStruct is a testing struct for table view
type TableStruct struct { //gti:add

	// an icon
	Icon icons.Icon

	// an integer field
	IntField int

	// a float field
	FloatField float32

	// a string field
	StrField string

	// a file
	File gi.FileName
}

// ILStruct is an inline-viewed struct
type ILStruct struct { //gti:add

	// click to show next
	On bool

	// can u see me?
	ShowMe string `viewif:"On"`

	// a conditional
	Cond int `viewif:"On"`

	// On and Cond=0 -- note that slbool as bool cannot be used directly..
	Cond1 string `viewif:"On&&Cond==0"`

	// if Cond=0
	Cond2 TableStruct `viewif:"On&&Cond<=1"`

	// a value
	Val float32
}

// Struct is a testing struct for struct view
type Struct struct { //gti:add

	// an enum
	Stripes gi.Stripes

	// a string
	Name string `viewif:"!(Stripes==[RowStripes,ColStripes])"`

	// click to show next
	ShowNext bool

	// can u see me?
	ShowMe string `viewif:"ShowNext"`

	// how about that
	Inline ILStruct `view:"inline"`

	// a conditional
	Cond int

	// if Cond=0
	Cond1 string `viewif:"Cond==0"`

	// if Cond=0
	Cond2 TableStruct `viewif:"Cond>=0"`

	// a value
	Val float32

	Things []*TableStruct

	Stuff []float32
}

func app() {
	tstslice := make([]string, 40)

	for i := 0; i < len(tstslice); i++ {
		tstslice[i] = fmt.Sprintf("el: %v", i)
	}

	tstmap := make(map[string]string)

	tstmap["mapkey1"] = "whatever"
	tstmap["mapkey2"] = "testing"
	tstmap["mapkey3"] = "boring"

	tsttable := make([]*TableStruct, 100)

	for i := range tsttable {
		ts := &TableStruct{IntField: i, FloatField: float32(i) / 10.0}
		tsttable[i] = ts
	}

	var stru Struct
	stru.Name = "happy"
	stru.Cond = 2
	stru.Val = 3.1415
	stru.Cond2.IntField = 22
	stru.Cond2.FloatField = 44.4
	stru.Cond2.StrField = "fi"
	// stru.Cond2.File = gi.FileName("views.go")
	stru.Things = make([]*TableStruct, 2)
	stru.Stuff = make([]float32, 3)

	// turn this on to see a trace of the rendering
	// gi.WinEventTrace = true
	// gi.RenderTrace = true
	// gi.LayoutTrace = true
	// gi.WinRenderTrace = true
	// gi.UpdateTrace = true
	// gi.KeyEventTrace = true

	gi.SetAppName("views")
	gi.SetAppAbout(`This is a demo of the MapView and SliceView views in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	sc := gi.NewScene("gogi-views-test").SetTitle("GoGi Views Test")

	trow := gi.NewLayout(sc, "trow").SetLayout(gi.LayoutHoriz)
	trow.Style(func(s *styles.Style) {
		s.AlignH = styles.AlignCenter
		s.AlignV = styles.AlignTop
		s.Margin.Set(units.Px(2))
		s.SetStretchMaxWidth()
	})

	gi.NewStretch(trow, "str1")

	but := gi.NewButton(trow, "slice-test").SetText("SliceDialog")
	but.Tooltip = "open a SliceViewDialog slice view with a lot of elments, for performance testing"
	but.OnClick(func(e events.Event) {
		sl := make([]float32, 2880)
		giv.SliceViewDialog(but, giv.DlgOpts{Title: "SliceView Test", Prompt: "It should open quickly."}, &sl, nil, nil).Run()
	})
	but = gi.NewButton(trow, "table-test").SetText("TableDialog")
	but.Tooltip = "open a TableViewDialog view "
	but.OnClick(func(e events.Event) {
		giv.TableViewDialog(but, giv.DlgOpts{Title: "TableView Test", Prompt: "how does it resize."}, &tsttable, nil, nil).Run()
	})

	lab1 := gi.NewLabel(trow, "lab1").SetText("<large>This is a test of the <tt>Slice</tt> and <tt>Map</tt> Views reflect-ive GUI</large>")
	lab1.Style(func(s *styles.Style) {
		s.SetStretchMaxWidth()
		s.Text.Align = styles.AlignCenter
	})
	gi.NewStretch(trow, "str2")

	spc := gi.NewSpace(sc, "spc1")
	spc.Style(func(s *styles.Style) {
		s.SetFixedHeight(units.Em(2))
	})

	split := gi.NewSplits(sc, "split")
	split.Dim = mat32.X

	// strv := giv.NewStructView(split, "strv")
	// strv.Sc = sc
	// strv.SetStruct(&stru)
	// strv.Style(func(s *styles.Style) {
	// 	s.SetStretchMax()
	// })

	// mv := giv.NewMapView(split, "mv")
	// mv.SetMap(&tstmap)
	// mv.Style(func(s *styles.Style) {
	// 	s.SetStretchMax()
	// })

	sv := giv.NewSliceView(split, "sv")
	// sv.SetState(true, states.ReadOnly)
	sv.SetSlice(&tstslice)
	sv.Style(func(s *styles.Style) {
		s.SetStretchMax()
	})

	tv := giv.NewTableView(split, "tv")
	// tv.SetState(true, states.ReadOnly)
	tv.SetSlice(&tsttable)
	tv.Style(func(s *styles.Style) {
		s.SetStretchMax()
	})

	// split.SetSplits(.3, .2, .2, .3)
	split.SetSplits(.5, .5)

	gi.NewWindow(sc).Run().Wait()
}
