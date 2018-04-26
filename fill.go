// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image/color"
	// "log"
	"github.com/rcoreilly/goki/ki/kit"
)

type FillRule int

const (
	FillRuleNonZero FillRule = iota
	FillRuleEvenOdd
	FillRuleN
)

//go:generate stringer -type=FillRule

var KiT_FillRule = kit.Enums.AddEnumAltLower(FillRuleN, false, StylePropProps, "FillRule")

// FillStyle contains all the properties specific to filling a region
type FillStyle struct {
	On      bool        `desc:"is fill active -- if property is none then false"`
	Color   Color       `xml:"fill" desc:"default fill color when such a color is needed -- Server could be anything"`
	Opacity float64     `xml:"fill-opacity" desc:"global alpha opacity / transparency factor"`
	Server  PaintServer `view:"-" desc:"paint server for the fill -- if solid color, defines fill color"`
	Rule    FillRule    `xml:"fill-rule" desc:"rule for how to fill more complex shapes with crossing lines"`
}

// initialize default values for paint fill
func (pf *FillStyle) Defaults() {
	pf.On = false // svg says fill is off by default
	pf.Color.SetColor(color.White)
	pf.Server = NewSolidcolorPaintServer(&pf.Color)
	pf.Rule = FillRuleNonZero
	pf.Opacity = 1.0
}

// need to do some updating after setting the style from user properties
func (pf *FillStyle) SetStylePost() {
	if pf.Color.IsNil() {
		pf.On = false
	} else {
		pf.On = true
		// for now -- todo: find a more efficient way of doing this, and only updating when necc
		pf.Server = NewSolidcolorPaintServer(&pf.Color)
		// todo: incorporate opacity
	}
}

func (pf *FillStyle) SetColor(cl *Color) {
	if cl == nil || cl.IsNil() {
		pf.On = false
	} else {
		pf.On = true
		pf.Color = *cl
		pf.Server = NewSolidcolorPaintServer(&pf.Color)
	}
}
