// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image/color"

	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

type FillRule int

const (
	FillRuleNonZero FillRule = iota
	FillRuleEvenOdd
	FillRuleN
)

//go:generate stringer -type=FillRule

var KiT_FillRule = kit.Enums.AddEnumAltLower(FillRuleN, false, StylePropProps, "FillRule")

func (ev FillRule) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *FillRule) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// FillStyle contains all the properties for filling a region
type FillStyle struct {
	On      bool      `desc:"is fill active -- if property is none then false"`
	Color   ColorSpec `xml:"fill" desc:"prop: fill = fill color specification"`
	Opacity float32   `xml:"fill-opacity" desc:"prop: fill-opacity = global alpha opacity / transparency factor"`
	Rule    FillRule  `xml:"fill-rule" desc:"prop: fill-rule = rule for how to fill more complex shapes with crossing lines"`
}

// Defaults initializes default values for paint fill
func (pf *FillStyle) Defaults() {
	pf.On = true // svg says fill is ON by default
	pf.SetColor(color.Black)
	pf.Rule = FillRuleNonZero
	pf.Opacity = 1.0
}

// SetStylePost does some updating after setting the style from user properties
func (pf *FillStyle) SetStylePost(props ki.Props) {
	if pf.Color.IsNil() {
		pf.On = false
	} else {
		pf.On = true
	}
}

// SetColor sets a solid fill color -- nil turns off filling
func (pf *FillStyle) SetColor(cl color.Color) {
	if cl == nil {
		pf.On = false
	} else {
		pf.On = true
		pf.Color.Color.SetColor(cl)
		pf.Color.Source = SolidColor
	}
}

// SetColorSpec sets full color spec from source
func (pf *FillStyle) SetColorSpec(cl *ColorSpec) {
	if cl == nil {
		pf.On = false
	} else {
		pf.On = true
		pf.Color.CopyFrom(cl)
	}
}
