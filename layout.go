// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "math"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"image"
)

// this is based on QtQuick layouts https://doc.qt.io/qt-5/qtquicklayouts-overview.html  https://doc.qt.io/qt-5/qml-qtquick-layouts-layout.html

// horizontal alignment type -- how to align items in the horizontal dimension
type AlignHorizontal int32

const (
	AlignLeft AlignHorizontal = iota
	AlignHCenter
	AlignRight
	AlignJustify
)

//go:generate stringer -type=AlignHorizontal

// vertical alignment type -- how to align items in the vertical dimension
type AlignVertical int32

const (
	AlignTop AlignVertical = iota
	AlignVCenter
	AlignBottom
	AlignBaseline
)

//go:generate stringer -type=AlignVertical

// todo: for style
// Align = layouts
// Content -- enum of various options
// Items -- similar enum -- combine
// Self "
// Flex -- flexbox -- https://www.w3schools.com/css/css3_flexbox.asp -- key to look at further for layout ideas
// Overflow is key for layout: visible, hidden, scroll, auto
// as is Position -- absolute, sticky, etc
// Resize: user-resizability
// vertical-align
// z-index

// style preferences on the layout of the element
type LayoutStyle struct {
	Width     units.Value   `xml:"width",desc:"specified size of element -- 0 if not specified"`
	Height    units.Value   `xml:"height",desc:"specified size of element -- 0 if not specified"`
	MaxWidth  units.Value   `xml:"max-width",desc:"specified maximum size of element -- 0 if not specified"`
	MaxHeight units.Value   `xml:"max-height",desc:"specified maximum size of element -- 0 if not specified"`
	MinWidth  units.Value   `xml:"min-width",desc:"specified mimimum size of element -- 0 if not specified"`
	MinHeight units.Value   `xml:"min-height",desc:"specified mimimum size of element -- 0 if not specified"`
	Offsets   []units.Value `xml:"{top,right,bottom,left}",desc:"specified offsets for each side"`
	Margin    units.Value   `xml:"margin",desc:"outer-most transparent space around box element"`
}

// size preferences -- a value of 0 indicates no preference
type SizePrefs struct {
	Min  Size2D `desc:"minimum size -- will not be less than this"`
	Pref Size2D `desc:"preferred size -- start here"`
	Max  Size2D `desc:"maximum size -- will not be greater than this -- 0 = max size"`
}

// want is the maximum across any of our prefs
func (sp *SizePrefs) Want() Size2D {
	want := sp.Max.Max(sp.Pref)
	want = want.Max(sp.Min)
	return want
}

// need is the minimum across any of our prefs
func (sp *SizePrefs) Need() Size2D {
	need := sp.Max.Min(sp.Pref)
	need = need.Min(sp.Min)
	return need
}

// 2D margins
type Margins struct {
	left, right, top, bottom float64
}

// set a single margin for all items
func (m *Margins) SetMargin(marg float64) {
	m.left = marg
	m.right = marg
	m.top = marg
	m.bottom = marg
}

// all the data needed to specify the layout of an item within a layout -- includes computed values of style prefs
type LayoutData struct {
	AlignH    AlignHorizontal `desc:"horizontal alignment"`
	AlignV    AlignVertical   `desc:"vertical alignment"`
	Size      SizePrefs       `desc:"size constraints for this item"`
	Margins   Margins         `desc:"margins around this item"`
	GridPos   image.Point     `desc:"position within a grid"`
	GridSpan  image.Point     `desc:"number of grid elements that we take up in each direction"`
	AllocPos  Point2D         `desc:"allocated relative position of this item, by the parent layout"`
	AllocSize Size2D          `desc:"allocated size of this item, by the parent layout"`
}

func (ld *LayoutData) Defaults() {
	if ld.GridSpan.X < 1 {
		ld.GridSpan.X = 1
	}
	if ld.GridSpan.Y < 1 {
		ld.GridSpan.Y = 1
	}
}

// called at start of layout process -- resets all values back to 0
func (ld *LayoutData) Reset() {
	ld.AllocPos = Point2DZero
	ld.AllocSize = Size2DZero
}

// get the effective position to use: if layout allocated, use that, otherwise user pos
func (ld *LayoutData) UsePos(userPos Point2D) {
	if ld.AllocPos.IsZero() {
		ld.AllocPos = userPos
	}
}

// get the effective size to use: if layout allocated, use that, otherwise user pos
func (ld *LayoutData) UseSize(userSize Size2D) {
	if ld.AllocSize.IsZero() {
		ld.AllocSize = userSize
	}
}

// want is max across prefs and existing allocsize
func (ld *LayoutData) WantSize() Size2D {
	want := ld.Size.Want()
	want = want.Max(ld.AllocSize)
	return want
}

// need is min across prefs and existing allocsize
func (ld *LayoutData) NeedSize() Size2D {
	need := ld.Size.Need()
	need = need.Max(ld.AllocSize)
	return need
}

// RowLayout arranges its elements in a horizontal fashion
type RowLayout struct {
	Node2DBase
	Layout LayoutData
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_RowLayout = ki.KiTypes.AddType(&RowLayout{})

func (g *RowLayout) GiNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *RowLayout) GiViewport2D() *Viewport2D {
	return nil
}

func (g *RowLayout) InitNode2D() {
}

func (g *RowLayout) Node2DBBox() image.Rectangle {
	return g.WinBBoxFromAlloc()
}

func (g *RowLayout) PaintProps2D() {
	g.PaintProps2DBase()
}

// need multiple iterations..
func (rl *RowLayout) Layout2D(iter int) {
	if len(rl.Children) == 0 {
		rl.Layout.AllocSize = rl.Layout.Size.Min
		return
	}

	// todo: need to include margins in all this!  do we use our margins or items?

	var sumWant, sumNeed, sumMin, maxWant, maxNeed, maxMin Size2D
	for _, c := range rl.Children {
		gii, ok := c.(Node2D)
		if !ok {
			continue
		}
		gi := gii.GiNode2D()
		want := gi.Layout.WantSize()
		need := gi.Layout.NeedSize()
		min := gi.Layout.Size.Need() // ignoring current allocations
		sumWant = sumWant.Add(want)
		sumNeed = sumNeed.Add(need)
		sumMin = sumMin.Add(min)
		maxWant = maxWant.Max(want)
		maxNeed = maxNeed.Max(need)
		maxMin = maxMin.Max(min)
	}

	minAvail := rl.Layout.Size.Need().X
	maxAvail := rl.Layout.Size.Want().X
	curAlloc := rl.Layout.AllocSize.X
	if curAlloc > 0 && curAlloc < maxAvail {
		maxAvail = curAlloc
	}
	if curAlloc > 0 && curAlloc < minAvail {
		minAvail = curAlloc
	}
	extra := 0.0
	avail := maxAvail // start with that
	targ := sumWant.X
	useWant := true
	useMin := false
	if avail == 0 { // no limits -- size to fit
		extra = 0.0
	} else {
		extra = avail - targ // go big first
		if extra < 0 {
			useWant = false
			targ = sumNeed.X
			extra = avail - targ
			if extra < 0 {
				useMin = true
				targ = sumMin.X
				extra = avail - targ
			}
		} else if avail-extra > minAvail { // lots of extra
			avail = minAvail
			extra = avail - targ
		}
	}

	// todo: vertical too!

	pos := 0.0
	per := extra / float64(len(rl.Children))
	for _, c := range rl.Children {
		gii, ok := c.(Node2D)
		if !ok {
			continue
		}
		gi := gii.GiNode2D()
		want := gi.Layout.WantSize()
		need := gi.Layout.NeedSize()
		min := gi.Layout.Size.Need() // ignoring current allocations
		base := need
		if useWant {
			base = want
		} else if useMin {
			base = min
		}
		gi.Layout.AllocSize = base
		gi.Layout.AllocSize.X += per
		gi.Layout.AllocPos.X = pos
		gi.Layout.AllocPos.Y = 0 // todo: alignment
		pos += gi.Layout.AllocSize.X
	}
}

func (g *RowLayout) Render2D() {
	g.GeomFromLayout()
}

func (g *RowLayout) CanReRender2D() bool {
	return false
}

// check for interface implementation
var _ Node2D = &RowLayout{}
