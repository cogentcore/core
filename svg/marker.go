// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/gi"
	"github.com/goki/ki/kit"
)

// Marker represents marker elements that can be drawn along paths (arrow heads, etc)
type Marker struct {
	SVGNodeBase
	RefPos    gi.Vec2D `xml:"{refX,refY}" desc:"reference position"`
	VertexPos gi.Vec2D `desc:"current vertex position -- this must be set by element that is using this marker prior to calling its render method"`
}

var KiT_Marker = kit.Types.AddType(&Marker{}, nil)

// MarkerUnits specifies units to use for svg marker elements
type MarkerUnits int32

const (
	StrokeWidth MarkerUnits = iota
	UserSpaceOnUse
	MarkerUnitsN
)

//go:generate stringer -type=MarkerUnits

var KiT_MarkerUnits = kit.Enums.AddEnumAltLower(MarkerUnitsN, false, gi.StylePropProps, "")

func (ev MarkerUnits) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *MarkerUnits) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }
