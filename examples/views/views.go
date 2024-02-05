// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate

import (
	"fmt"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/mat32"
)

// TableStruct is a testing struct for table view
type TableStruct struct { //gti:add

	// an icon
	Icon icons.Icon

	// an integer field
	IntField int `default:"2"`

	// a float field
	FloatField float32

	// a string field
	StrField string

	// a file
	File gi.Filename
}

// ILStruct is an inline-viewed struct
type ILStruct struct { //gti:add

	// click to show next
	On bool

	// can u see me?
	ShowMe string

	// a conditional
	Cond int

	// On and Cond=0
	Cond1 string

	// if Cond=0
	Cond2 TableStruct

	// a value
	Val float32
}

func (il *ILStruct) ShouldShow(field string) bool {
	switch field {
	case "ShowMe", "Cond":
		return il.On
	case "Cond1":
		return il.On && il.Cond == 0
	case "Cond2":
		return il.On && il.Cond <= 1
	}
	return true
}

// Struct is a testing struct for struct view
type Struct struct { //gti:add

	// An enum value
	Enum gi.ButtonTypes

	// a string
	Name string

	// click to show next
	ShowNext bool

	// can u see me?
	ShowMe string

	// how about that
	Inline ILStruct

	// a conditional
	Cond int

	// if Cond=0
	Cond1 string

	// if Cond>=0
	Cond2 TableStruct

	// a value
	Val float32

	Vec mat32.Vec2

	Things []*TableStruct

	Stuff []float32
}

func (st *Struct) ShouldShow(field string) bool {
	switch field {
	case "Name":
		return st.Enum <= gi.ButtonElevated
	case "ShowMe":
		return st.ShowNext
	case "Cond1":
		return st.Cond == 0
	case "Cond2":
		return st.Cond >= 0
	}
	return true
}

func main() {
	tstslice := make([]string, 20)

	for i := 0; i < len(tstslice); i++ {
		tstslice[i] = fmt.Sprintf("el: %v", i)
	}
	tstslice[10] = "this is a particularly long slice value"

	tstmap := make(map[string]string)

	tstmap["mapkey1"] = "whatever"
	tstmap["mapkey2"] = "testing"
	tstmap["mapkey3"] = "boring"

	tsttable := make([]*TableStruct, 100)

	for i := range tsttable {
		ts := &TableStruct{IntField: i, FloatField: float32(i) / 10.0}
		tsttable[i] = ts
	}

	tsttable[0].StrField = "this is a particularly long field"

	var stru Struct
	stru.Name = "happy"
	stru.Cond = 2
	stru.Val = 3.1415
	stru.Vec.Set(5, 7)
	stru.Inline.Val = 3
	stru.Cond2.IntField = 22
	stru.Cond2.FloatField = 44.4
	stru.Cond2.StrField = "fi"
	// stru.Cond2.File = gi.Filename("views.go")
	stru.Things = make([]*TableStruct, 2)
	stru.Stuff = make([]float32, 3)

	b := gi.NewBody("Cogent Core Views Demo")

	b.AddAppBar(func(tb *gi.Toolbar) {
		gi.NewButton(tb, "slice-test").SetText("SliceDialog").
			SetTooltip("open a SliceViewDialog slice view with a lot of elments, for performance testing").
			OnClick(func(e events.Event) {
				sl := make([]float32, 2880)
				d := gi.NewBody().AddTitle("SliceView Test").AddText("It should open quickly.")
				giv.NewSliceView(d).SetSlice(&sl)
				d.NewFullDialog(tb).Run()
			})
		gi.NewButton(tb, "table-test").SetText("TableDialog").
			SetTooltip("open a TableViewDialog view").
			OnClick(func(e events.Event) {
				d := gi.NewBody().AddTitle("TableView Test").AddText("how does it resize.")
				giv.NewTableView(d).SetSlice(&tsttable)
				d.NewFullDialog(tb).Run()
			})
	})

	// split := gi.NewSplits(b, "split")
	// split.Dim = mat32.X
	// split.SetSplits(.3, .2, .2, .3)
	// split.SetSplits(.5, .5)

	ts := gi.NewTabs(b)
	tst := ts.NewTab("StructView")
	tmv := ts.NewTab("MapView")
	tsl := ts.NewTab("SliceView")
	ttv := ts.NewTab("TableView")
	_, _, _, _ = tst, tmv, tsl, ttv

	strv := giv.NewStructView(tst, "strv")
	strv.SetStruct(&stru)

	mv := giv.NewMapView(tmv, "mv")
	mv.SetMap(&tstmap)

	sv := giv.NewSliceView(tsl, "sv")
	// sv.SetState(true, states.ReadOnly)
	sv.SetSlice(&tstslice)

	tv := giv.NewTableView(ttv, "tv")
	// tv.SetState(true, states.ReadOnly)
	tv.SetSlice(&tsttable)

	b.RunMainWindow()
}
