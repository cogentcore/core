// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/gi/gist"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

////////////////////////////////////////////////////////////////////////////////////////
// ViewBox defines the SVG viewbox

// ViewBox is used in SVG to define the coordinate system
type ViewBox struct {
	Min                 mat32.Vec2                 `desc:"offset or starting point in parent Viewport2D"`
	Size                mat32.Vec2                 `desc:"size of viewbox within parent Viewport2D"`
	PreserveAspectRatio ViewBoxPreserveAspectRatio `desc:"how to scale the view box within parent Viewport2D"`
}

// todo: need to implement the viewbox preserve aspect ratio logic!

// Defaults returns viewbox to defaults
func (vb *ViewBox) Defaults() {
	vb.Min = mat32.Vec2Zero
	vb.Size = mat32.Vec2Zero
	vb.PreserveAspectRatio.Align = NoAlign
	vb.PreserveAspectRatio.MeetOrSlice = Meet
}

// todo: these should be regular ints and use bitflag etc.

// ViewBoxAlign defines values for the PreserveAspectRatio alignment factor
type ViewBoxAlign int32

const (
	NoAlign ViewBoxAlign = 1 << iota          // do not preserve uniform scaling
	XMin                                      // align ViewBox.Min with smallest values of Viewport
	XMid                                      // align ViewBox.Min with midpoint values of Viewport
	XMax                                      // align ViewBox.Min+Size with maximum values of Viewport
	XMask   ViewBoxAlign = XMin + XMid + XMax // mask for X values -- clear all X before setting new one
	YMin    ViewBoxAlign = 1 << iota          // align ViewBox.Min with smallest values of Viewport
	YMid                                      // align ViewBox.Min with midpoint values of Viewport
	YMax                                      // align ViewBox.Min+Size with maximum values of Viewport
	YMask   ViewBoxAlign = YMin + YMid + YMax // mask for Y values -- clear all Y before setting new one
)

// ViewBoxMeetOrSlice defines values for the PreserveAspectRatio meet or slice factor
type ViewBoxMeetOrSlice int32

var KiT_ViewBoxMeetOrSlice = kit.Enums.AddEnumAltLower(ViewBoxMeetOrSliceN, kit.NotBitFlag, gist.StylePropProps, "")

func (ev ViewBoxMeetOrSlice) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *ViewBoxMeetOrSlice) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

const (
	// Meet means the entire ViewBox is visible within Viewport, and it is
	// scaled up as much as possible to meet the align constraints
	Meet ViewBoxMeetOrSlice = iota

	// Slice means the entire ViewBox is covered by the ViewBox, and the
	// ViewBox is scaled down as much as possible, while still meeting the
	// align constraints
	Slice

	ViewBoxMeetOrSliceN
)

//go:generate stringer -type=ViewBoxMeetOrSlice

// ViewBoxPreserveAspectRatio determines how to scale the view box within parent Viewport2D
type ViewBoxPreserveAspectRatio struct {
	Align       ViewBoxAlign       `svg:"align" desc:"how to align x,y coordinates within viewbox"`
	MeetOrSlice ViewBoxMeetOrSlice `svg:"meetOrSlice" desc:"how to scale the view box relative to the viewport"`
}
