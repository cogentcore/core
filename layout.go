// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "math"
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

// size preferences -- a value of 0 indicates no preference
type SizePrefs struct {
	Min  Size2D `desc:"minimum size -- will not be less than this"`
	Pref Size2D `desc:"preferred size -- start here"`
	Max  Size2D `desc:"maximum size -- will not be greater than this -- 0 = max size"`
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

// all the data needed to specify the layout of an item within a layout
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
	ld.GridSpan = image.Point{1, 1}
}

// RowLayout arranges its elements in a horizontal fashion
type RowLayout struct {
	WidgetBase
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_RowLayout = ki.KiTypes.AddType(&RowLayout{})

func (rl *RowLayout) DoLayout(vp *Viewport2D) {

}
