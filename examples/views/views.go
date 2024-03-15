// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate

import (
	"fmt"

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
	Inline ILStruct `view:"inline"`

	// a conditional
	Cond int

	// if Cond=0
	Cond1 string

	// if Cond>=0
	Cond2 TableStruct

	// a value
	Val float32

	Vec mat32.Vec2

	Things []TableStruct

	Stuff []float32

	// a file
	File gi.Filename
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

	str := Struct{
		Name:   "happy",
		Cond:   2,
		Val:    3.1415,
		Vec:    mat32.V2(5, 7),
		Inline: ILStruct{Val: 3},
		Cond2: TableStruct{
			IntField:   22,
			FloatField: 44.4,
			StrField:   "fi",
			File:       "views.go",
		},
		Things: make([]TableStruct, 2),
		Stuff:  make([]float32, 3),
	}

	b := gi.NewBody("Cogent Core Views Demo")

	ts := gi.NewTabs(b)

	giv.NewStructView(ts.NewTab("Struct view")).SetStruct(&str)
	giv.NewMapView(ts.NewTab("Map view")).SetMap(&tstmap)
	giv.NewSliceView(ts.NewTab("Slice view")).SetSlice(&tstslice)
	giv.NewTableView(ts.NewTab("Table view")).SetSlice(&tsttable)

	b.RunMainWindow()
}
