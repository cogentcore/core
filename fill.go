// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image/color"
	"log"
)

type FillRule int

const (
	FillRuleNonZero FillRule = iota
	FillRuleEvenOdd
)

//go:generate stringer -type=FillRule

// FillPaint contains all the properties specific to filling a region
type FillPaint struct {
	On     bool        `desc:"is fill active -- if property is none then false"`
	Color  Color       `desc:"default fill color when such a color is needed -- Server could be anything"`
	Server PaintServer `svg:"fill",desc:"paint server for the fill -- if solid color, defines fill color"`
	Rule   FillRule    `svg:"fill-rule",desc:"rule for how to fill more complex shapes with crossing lines"`
}

// initialize default values for paint fill
func (pf *FillPaint) Defaults() {
	pf.On = false // svg says fill is off by default
	pf.Color.SetColor(color.White)
	pf.Server = NewSolidcolorPaintServer(&pf.Color)
	pf.Rule = FillRuleNonZero
}

// todo: figure out more elemental, generic de-stringer kind of thing

// update the fill settings from the style info on the node
func (pf *FillPaint) SetFromNode(g *Node2DBase) {
	// always check if property has been set before setting -- otherwise defaults to empty -- true = inherit props
	// todo: need to be able to process colors!

	if val, got := g.PropColor("fill"); got { // todo: support url to other paint server types
		if val == nil {
			pf.On = false
		} else {
			pf.On = true
			pf.Color = *val
			pf.Server = NewSolidcolorPaintServer(val)
		}
	}
	if _, got := g.PropNumber("fill-opacity"); got {
		// todo: need to set the color alpha according to value
	}
	if val, got := g.PropEnum("fill-rule"); got {
		var fr FillRule = -1
		switch val {
		case "nonzero":
			fr = FillRuleNonZero
		case "evenodd":
			fr = FillRuleEvenOdd
		}
		if fr == -1 {
			i, err := StringToFillRule(val) // stringer gen
			if err != nil {
				pf.Rule = i
			} else {
				log.Print(err)
			}
		}
	}
}
