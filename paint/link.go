// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"

	"goki.dev/girl/styles"
	"goki.dev/mat32/v2"
)

// TextLink represents a hyperlink within rendered text
type TextLink struct {

	// text label for the link
	Label string

	// full URL for the link
	URL string

	// Style for rendering this link, set by the controlling widget
	Style styles.FontRender

	// additional properties defined for the link, from the parsed HTML attributes
	Props map[string]any

	// span index where link starts
	StartSpan int

	// index in StartSpan where link starts
	StartIdx int

	// span index where link ends (can be same as EndSpan)
	EndSpan int

	// index in EndSpan where link ends (index of last rune in label)
	EndIdx int
}

// Bounds returns the bounds of the link
func (tl *TextLink) Bounds(tr *Text, pos mat32.Vec2) image.Rectangle {
	stsp := &tr.Spans[tl.StartSpan]
	tpos := pos.Add(stsp.RelPos)
	sr := &(stsp.Render[tl.StartIdx])
	sp := tpos.Add(sr.RelPos)
	sp.Y -= sr.Size.Y
	ep := sp
	if tl.EndSpan == tl.StartSpan {
		er := &(stsp.Render[tl.EndIdx])
		ep = tpos.Add(er.RelPos)
		ep.X += er.Size.X
	} else {
		er := &(stsp.Render[len(stsp.Render)-1])
		ep = tpos.Add(er.RelPos)
		ep.X += er.Size.X
	}
	return image.Rectangle{Min: sp.ToPointFloor(), Max: ep.ToPointCeil()}
}
