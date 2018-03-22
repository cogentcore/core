// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"image"
	"math"
)

// this is based on QtQuick layouts https://doc.qt.io/qt-5/qtquicklayouts-overview.html  https://doc.qt.io/qt-5/qml-qtquick-layouts-layout.html

// horizontal alignment type -- how to align items in the horizontal dimension
type AlignHoriz int32

const (
	AlignLeft AlignHoriz = iota
	AlignHCenter
	AlignRight
	AlignHJustify
	AlignHorizN
)

//go:generate stringer -type=AlignHoriz

var KiT_AlignHoriz = ki.Enums.AddEnumAltLower(AlignLeft, false, nil, "Align", int64(AlignHorizN))

// vertical alignment type -- how to align items in the vertical dimension -- must correspond with horizontal for layout
type AlignVert int32

const (
	AlignTop AlignVert = iota
	AlignVCenter
	AlignBottom
	AlignVJustify
	AlignBaseline
	AlignVertN
)

var KiT_AlignVert = ki.Enums.AddEnumAltLower(AlignTop, false, nil, "Align", int64(AlignVertN))

//go:generate stringer -type=AlignVert

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
	z_index   int           `xml:"z-index",desc:"ordering factor for rendering depth -- lower numbers rendered first -- sort children according to this factor"`
	AlignH    AlignHoriz    `xml:"align-horiz",desc:"horizontal alignment -- for widget layouts -- not a standard css property"`
	AlignV    AlignVert     `xml:"align-vert",desc:"vertical alignment -- for widget layouts -- not a standard css property"`
	PosX      units.Value   `xml:"x",desc:"horizontal position -- often superceded by layout but otherwise used"`
	PosY      units.Value   `xml:"y",desc:"vertical position -- often superceded by layout but otherwise used"`
	Width     units.Value   `xml:"width",desc:"specified size of element -- 0 if not specified"`
	Height    units.Value   `xml:"height",desc:"specified size of element -- 0 if not specified"`
	MaxWidth  units.Value   `xml:"max-width",desc:"specified maximum size of element -- 0  means just use other values, negative means stretch"`
	MaxHeight units.Value   `xml:"max-height",desc:"specified maximum size of element -- 0 means just use other values, negative means stretch"`
	MinWidth  units.Value   `xml:"min-width",desc:"specified mimimum size of element -- 0 if not specified"`
	MinHeight units.Value   `xml:"min-height",desc:"specified mimimum size of element -- 0 if not specified"`
	Offsets   []units.Value `xml:"{top,right,bottom,left}",desc:"specified offsets for each side"`
	Margin    units.Value   `xml:"margin",desc:"outer-most transparent space around box element -- todo: can be specified per side"`
}

func (ls *LayoutStyle) Defaults() {
	ls.MinWidth.Set(1.0, units.Em)
	ls.MinHeight.Set(1.0, units.Em)
	ls.Width.Set(1.0, units.Em)
	ls.Height.Set(1.0, units.Em)
}

func (ls *LayoutStyle) SetStylePost() {
}

// return the alignment for given dimension, using horiz terminology (top = left, etc)
func (ls *LayoutStyle) AlignDim(d Dims2D) AlignHoriz {
	switch d {
	case X:
		return ls.AlignH
	default:
		return AlignHoriz(ls.AlignV)
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Layout Data for actually computing the layout

// size preferences
type SizePrefs struct {
	Need Vec2D `desc:"minimum size needed -- set to at least computed allocsize"`
	Pref Vec2D `desc:"preferred size -- start here for layout"`
	Max  Vec2D `desc:"maximum size -- will not be greater than this -- 0 = no constraint, neg = stretch"`
}

// return true if Max < 0 meaning can stretch infinitely along given dimension
func (sp SizePrefs) HasMaxStretch(d Dims2D) bool {
	return (sp.Max.Dim(d) < 0.0)
}

// return true if Pref > Need meaning can stretch more along given dimension
func (sp SizePrefs) CanStretchNeed(d Dims2D) bool {
	return (sp.Pref.Dim(d) > sp.Need.Dim(d))
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

// all the data needed to specify the layout of an item within a layout -- includes computed values of style prefs -- everything is concrete and specified here, whereas style may not be fully resolved
type LayoutData struct {
	Size      SizePrefs   `desc:"size constraints for this item -- from layout style"`
	Margins   Margins     `desc:"margins around this item"`
	GridPos   image.Point `desc:"position within a grid"`
	GridSpan  image.Point `desc:"number of grid elements that we take up in each direction"`
	AllocPos  Vec2D       `desc:"allocated relative position of this item, by the parent layout"`
	AllocSize Vec2D       `desc:"allocated size of this item, by the parent layout"`
}

func (ld *LayoutData) Defaults() {
	if ld.GridSpan.X < 1 {
		ld.GridSpan.X = 1
	}
	if ld.GridSpan.Y < 1 {
		ld.GridSpan.Y = 1
	}
}

func (ld *LayoutData) SetFromStyle(ls *LayoutStyle) {
	ld.Reset()
	// these are layout hints:
	ld.Size.Need = Vec2D{ls.MinWidth.Dots, ls.MinHeight.Dots}
	ld.Size.Pref = Vec2D{ls.Width.Dots, ls.Height.Dots}
	ld.Size.Max = Vec2D{ls.MaxWidth.Dots, ls.MaxHeight.Dots}

	// this is an actual initial desired setting
	ld.AllocPos = Vec2D{ls.PosX.Dots, ls.PosY.Dots}
	// not setting size, so we can keep that as a separate constraint
}

// called at start of layout process -- resets all values back to 0
func (ld *LayoutData) Reset() {
	ld.AllocPos = Vec2DZero
	ld.AllocSize = Vec2DZero
}

// update our sizes based on AllocSize and Max constraints, etc
func (ld *LayoutData) UpdateSizes() {
	ld.Size.Need.SetMax(ld.AllocSize)   // min cannot be < alloc -- bare min
	ld.Size.Pref.SetMax(ld.Size.Need)   // pref cannot be < min
	ld.Size.Need.SetMinPos(ld.Size.Max) // min cannot be > max
	ld.Size.Pref.SetMinPos(ld.Size.Max) // pref cannot be > max
}

////////////////////////////////////////////////////////////////////////////////////////
//    Layout handles all major types of layout

// different types of layouts
type Layouts int32

const (
	// arrange items horizontally across a row
	LayoutRow Layouts = iota
	// arrange items vertically in a column
	LayoutCol
	// arrange items according to a grid
	LayoutGrid
	// arrange items horizontally across a row, overflowing vertically as needed
	LayoutRowFlow
	// arrange items vertically within a column, overflowing horizontally as needed
	LayoutColFlow
	LayoutsN
)

//go:generate stringer -type=Layouts

// note: Layout cannot be a Widget type because Controls in Widget is a Layout..

// Layout is the primary node type responsible for organizing the sizes and
// positions of child widgets -- all arbitrary collections of widgets should
// generally be contained within a layout -- otherwise the parent widget must
// take over responsibility for positioning.  The alignment is NOT inherited
// by default so must be specified per child, except that the parent alignment
// is used within the relevant dimension (e.g., align-horiz for a LayoutRow
// layout, to determine left, right, center, justified)
type Layout struct {
	Node2DBase
	// type of layout to use
	Layout Layouts
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Layout = ki.Types.AddType(&Layout{}, nil)

// do we sum up elements along given dimension?  else max
func (ly *Layout) SumDim(d Dims2D) bool {
	if (d == X && ly.Layout == LayoutRow) || (d == Y && ly.Layout == LayoutCol) {
		return true
	}
	return false
}

// first depth-first pass: terminal concrete items compute their AllocSize
// we focus on Need: Max(Min, AllocSize), and Want: Max(Pref, AllocSize) -- Max is
// only used if we need to fill space, during final allocation
//
// second me-first pass: each layout allocates AllocSize for its children based on
// aggregated size data, and so on down the tree

// first pass: gather the size information from the children
func (ly *Layout) GatherSizes() {
	if len(ly.Children) == 0 {
		return
	}

	var sumPref, sumNeed, maxPref, maxNeed Vec2D
	for _, c := range ly.Children {
		_, gi := KiToNode2D(c)
		if gi == nil {
			continue
		}
		// fmt.Printf("child %v lay size alloc: %v\n", gi.Name, gi.LayData.AllocSize)
		gi.LayData.UpdateSizes()
		// fmt.Printf("child %v lay size need: %v pref: %v\n", gi.Name, gi.LayData.Size.Need, gi.LayData.Size.Pref)
		sumNeed = sumNeed.Add(gi.LayData.Size.Need)
		sumPref = sumPref.Add(gi.LayData.Size.Pref)
		maxNeed = maxNeed.Max(gi.LayData.Size.Need)
		maxPref = maxPref.Max(gi.LayData.Size.Pref)
	}

	for d := X; d <= Y; d++ {
		if ly.SumDim(d) { // our layout now updated to sum
			ly.LayData.Size.Need.SetMaxDim(d, sumNeed.Dim(d))
			ly.LayData.Size.Pref.SetMaxDim(d, sumPref.Dim(d))
		} else { // use max for other dir
			ly.LayData.Size.Need.SetMaxDim(d, maxNeed.Dim(d))
			ly.LayData.Size.Pref.SetMaxDim(d, maxPref.Dim(d))
		}
	}
	ly.LayData.Size.Need.SetAddVal(2.0 * ly.Style.Layout.Margin.Dots)
	ly.LayData.Size.Pref.SetAddVal(2.0 * ly.Style.Layout.Margin.Dots)

	// todo: something entirely different needed for grids..

	ly.LayData.UpdateSizes() // enforce max and normal ordering, etc
}

// in case we don't have any explicit allocsize set for us -- go up parents
// until we find one -- typically a viewport
func (ly *Layout) AllocFromParent() {
	if !ly.LayData.AllocSize.IsZero() {
		return
	}

	ly.FunUpParent(0, ly.This, func(k ki.Ki, level int, d interface{}) bool {
		_, pg := KiToNode2D(k)
		if pg == nil {
			return false
		}
		if !pg.LayData.AllocSize.IsZero() {
			ly.LayData.AllocSize = pg.LayData.AllocSize
			fmt.Printf("layout got parent alloc: %v from %v\n", ly.LayData.AllocSize,
				pg.Name)
			return false
		}
		return true
	})
}

// parent has now set our AllocSize -- we update AllocSize for our children
// fixed layout case along either horiz or vert dim -- although
// align horiz and vert are used here, they should just be cast from other if
// dim == Y
func (ly *Layout) LayoutFixed(dim Dims2D) {
	sz := len(ly.Children)
	if sz == 0 {
		return
	}

	othDim := OtherDim(dim)
	alDim := ly.Style.Layout.AlignDim(dim)

	marg := 2.0 * ly.Style.Layout.Margin.Dots

	avail := ly.LayData.AllocSize.SubVal(marg)
	targ := ly.LayData.Size.Pref.SubVal(marg)
	usePref := [2]bool{true, true}
	extra := avail.Sub(targ)
	fmt.Printf("first extra: %v avail: \n", extra.Dim(dim))
	for d := X; d <= Y; d++ {
		if extra.Dim(d) < -0.1 { // not fitting in pref, go with min
			usePref[d] = false
			targ.SetDim(d, ly.LayData.Size.Need.Dim(d)-marg)
			extra.SetDim(d, avail.Dim(d)-targ.Dim(d))
		}
	}
	extra.SetMaxVal(0.0) // no negatives

	extraPer := Vec2DZero
	nstretch := 0
	stretchNeed := false                       // stretch relative to need
	stretchMax := false                        // only stretch Max = neg
	addSpace := false                          // apply extra toward spacing -- for justify
	if usePref[dim] && extra.Dim(dim) >= 0.0 { // have some stretch extra
		for _, c := range ly.Children {
			_, gi := KiToNode2D(c)
			if gi == nil {
				continue
			}
			if gi.LayData.Size.HasMaxStretch(dim) { // negative = stretch
				nstretch++
			}
		}
		if nstretch > 0 {
			stretchMax = true // only stretch those marked as infinitely stretchy
			extraPer.SetDim(dim, extra.Dim(dim)/float64(nstretch))
		}
	} else if extra.Dim(dim) >= 0.0 { // extra relative to Need
		for _, c := range ly.Children {
			_, gi := KiToNode2D(c)
			if gi == nil {
				continue
			}
			if gi.LayData.Size.HasMaxStretch(dim) || gi.LayData.Size.CanStretchNeed(dim) {
				nstretch++
			}
		}
		if nstretch > 0 {
			stretchNeed = true // stretch relative to need
			extraPer.SetDim(dim, extra.Dim(dim)/float64(nstretch))
		}
	}
	fmt.Printf("extra: %v, exper: %v, nstretch: %v, stretchMax: %v, stretchNeed: %v, alDim: %v\n", extra.Dim(dim), extraPer.Dim(dim), nstretch, stretchMax, stretchNeed, alDim)

	if sz > 1 && extra.Dim(dim) > 0.0 && alDim == AlignHJustify && !stretchNeed && !stretchMax {
		addSpace = true
		// if neither, then just distribute as spacing for justify
		extraPer.SetDim(dim, extra.Dim(dim)/float64(sz-1))
	}

	// now arrange everyone
	pos := Vec2DZero.AddVal(ly.Style.Layout.Margin.Dots)

	// todo: need a direction setting too
	if alDim == AlignRight && !stretchNeed && !stretchMax {
		pos.SetDim(dim, extra.Dim(dim)) // start with all the extra space
	}

	for i, c := range ly.Children {
		_, gi := KiToNode2D(c)
		if gi == nil {
			continue
		}
		fmt.Printf("child %v lay size need: %v pref: %v\n", gi.Name, gi.LayData.Size.Need, gi.LayData.Size.Pref)

		base := gi.LayData.Size.Need
		for d := X; d <= Y; d++ {
			if usePref[d] {
				base.SetDim(d, gi.LayData.Size.Pref.Dim(d))
			} else if d != dim { // in other dim, use everything we've got
				base.SetDim(d, avail.Dim(d))
			}
		}
		gi.LayData.AllocSize = base
		if stretchMax { // negative = stretch
			if gi.LayData.Size.HasMaxStretch(dim) {
				gi.LayData.AllocSize.SetDim(dim, base.Dim(dim)+extraPer.Dim(dim))
			}
		} else if stretchNeed {
			if gi.LayData.Size.HasMaxStretch(dim) || gi.LayData.Size.CanStretchNeed(dim) {
				gi.LayData.AllocSize.SetDim(dim, base.Dim(dim)+extraPer.Dim(dim))
			}
		} else if addSpace { // implies align justify
			if i > 0 {
				pos.SetDim(dim, pos.Dim(dim)+extraPer.Dim(dim))
			}
		}

		// position along the other dimension
		// extra room for positioning
		ex := avail.Dim(othDim) - gi.LayData.AllocSize.Dim(othDim)
		ex = math.Max(ex, 0.0) // gt 0

		switch gi.Style.Layout.AlignDim(othDim) {
		case AlignLeft:
			pos.SetDim(othDim, 0.0)
		case AlignHCenter:
			pos.SetDim(othDim, 0.5*ex)
		case AlignRight:
			pos.SetDim(othDim, ex)
		case AlignHJustify: // nonsensical, just top / left
			pos.SetDim(othDim, 0.0)
		}

		gi.LayData.AllocPos = pos
		fmt.Printf("child %v lay size: %v pos: %v\n", gi.Name, gi.LayData.AllocSize, gi.LayData.AllocPos)
		pos.SetDim(dim, pos.Dim(dim)+gi.LayData.AllocSize.Dim(dim))
	}
}

func (ly *Layout) AsNode2D() *Node2DBase {
	return &ly.Node2DBase
}

func (ly *Layout) AsViewport2D() *Viewport2D {
	return nil
}

func (ly *Layout) InitNode2D() {
}

func (ly *Layout) Node2DBBox() image.Rectangle {
	return ly.WinBBoxFromAlloc()
}

func (ly *Layout) Style2D() {
	ly.Style2DWidget()
}

// need multiple iterations?
func (ly *Layout) Layout2D(iter int) {

	if iter == 0 {
		ly.InitLayout2D()
		ly.GatherSizes()
	} else {
		ly.AllocFromParent() // in case we didn't get anything
		switch ly.Layout {
		case LayoutRow:
			ly.LayoutFixed(X)
		case LayoutCol:
			ly.LayoutFixed(Y)
		}
		ly.GeomFromLayout()
	}
	// todo: test if this is needed -- if there are any el-relative settings anyway
	ly.Style.SetUnitContext(&ly.Viewport.Render, 0)
}

func (ly *Layout) Render2D() {

}

func (ly *Layout) CanReRender2D() bool {
	return false
}

func (ly *Layout) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Layout{}

///////////////////////////////////////////////////////////
//    Frame -- generic container

// Frame is a basic container for widgets -- a layout that renders the
// standard box model
type Frame struct {
	Layout
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Frame = ki.Types.AddType(&Frame{}, nil)

func (g *Frame) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Frame) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Frame) InitNode2D() {
}

var FrameProps = map[string]interface{}{
	"border-width":     "1px",
	"border-radius":    "0px",
	"border-color":     "black",
	"border-style":     "solid",
	"padding":          "2px",
	"margin":           "2px",
	"color":            "black",
	"background-color": "#FFF",
}

func (g *Frame) Style2D() {
	// first do our normal default styles
	g.Style.SetStyle(nil, &StyleDefault, FrameProps)
	// then style with user props
	g.Style2DWidget()
}

func (g *Frame) Layout2D(iter int) {
	// todo:
	// g.Layout.Layout2D(iter)
	if iter == 0 {
		g.InitLayout2D()
	} else {
		g.GeomFromLayout()
	}
	// todo: test for use of parent-el relative units -- indicates whether multiple loops
	// are required
	g.Style.SetUnitContext(&g.Viewport.Render, 0)
}

func (g *Frame) Node2DBBox() image.Rectangle {
	return g.WinBBoxFromAlloc()
}

func (g *Frame) Render2D() {
	pc := &g.Paint
	st := &g.Style
	rs := &g.Viewport.Render
	pc.Stroke.SetColor(&st.Border.Color)
	pc.Stroke.Width = st.Border.Width
	pc.Fill.SetColor(&st.Background.Color)
	pos := g.LayData.AllocPos.AddVal(st.Layout.Margin.Dots).SubVal(st.Border.Width.Dots)
	sz := g.LayData.AllocSize.SubVal(2.0 * st.Layout.Margin.Dots).AddVal(2.0 * st.Border.Width.Dots)
	// pos := g.LayData.AllocPos
	// sz := g.LayData.AllocSize
	rad := st.Border.Radius.Dots
	if rad == 0.0 {
		pc.DrawRectangle(rs, pos.X, pos.Y, sz.X, sz.Y)
	} else {
		pc.DrawRoundedRectangle(rs, pos.X, pos.Y, sz.X, sz.Y, rad)
	}
	pc.FillStrokeClear(rs)
}

func (g *Frame) CanReRender2D() bool {
	return true
}

func (g *Frame) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Frame{}
