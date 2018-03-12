// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/ki"
	"image"
	"log"
	"strings"
	"unicode"
)

// todo: needs to include justify https://www.w3.org/TR/css-text-3/#text-align-property
// how to align text
type TextAlign int

const (
	TextAlignLeft TextAlign = iota
	TextAlignCenter
	TextAlignRight
)

//go:generate stringer -type=TextAlign

// text layout information for painter
type PaintTextLayout struct {
	Wrap    bool      `svg:"word-wrap",desc:"wrap text within a given size specified in Size.X"`
	Align   TextAlign `svg:"text-align",desc:"how to align text"`
	Spacing Size2D    `svg:"{dx,dy}",desc:"spacing between characters and lines"`
}

func (p *PaintTextLayout) Defaults() {
	p.Wrap = false
	p.Align = TextAlignLeft
	p.Spacing = Size2D{1.0, 1.0}
}

// update the font settings from the style info on the node
func (pt *PaintTextLayout) SetFromNode(g *Node2DBase) {
	// always check if property has been set before setting -- otherwise defaults to empty -- true = inherit props

	if wr, got := g.GiPropBool("word-wrap"); got { // gi version
		pt.Wrap = wr
	}
	if sz, got := g.PropNumber("text-spacing"); got {
		pt.Spacing.Y = sz
	}
	if es, got := g.PropEnum("text-align"); got {
		var al TextAlign = -1
		switch es { // first go through short-hand codes
		case "left":
			al = TextAlignLeft
		case "start":
			al = TextAlignLeft
		case "center":
			al = TextAlignCenter
		case "right":
			al = TextAlignRight
		case "end":
			al = TextAlignRight
		}
		if al == -1 {
			i, err := StringToTextAlign(es) // stringer gen
			if err != nil {
				pt.Align = i
			} else {
				log.Print(err)
			}
		} else {
			pt.Align = al
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Text2D Node

// todo: lots of work likely needed on laying-out text in proper way
// https://www.w3.org/TR/SVG2/text.html#GlyphsMetrics
// todo: tspan element

// 2D Text
type Text2D struct {
	Node2DBase
	Text        string   `svg:"text",desc:"text string to render"`
	WrappedText []string `json:"-","desc:word-wrapped version of the string"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Text2D = ki.KiTypes.AddType(&Text2D{})

func (g *Text2D) GiNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Text2D) GiViewport2D() *Viewport2D {
	return nil
}

func (g *Text2D) InitNode2D() {
	g.Layout.Defaults()
}

func (g *Text2D) PaintProps2D() {
	// pc := &g.MyPaint
	// if pc.HasNoStrokeOrFill() || len(g.Text) == 0 {
	// 	pc.Off = true
	// }
}

func (g *Text2D) Layout2D(iter int) {
	if iter == 0 {
		pc := &g.MyPaint
		var w, h float64
		// pre-wrap the text
		if pc.TextLayout.Wrap {
			g.WrappedText, h = pc.MeasureStringWrapped(g.Text, g.Size.X, pc.TextLayout.Spacing.Y)
		} else {
			w, h = pc.MeasureString(g.Text)
		}
		g.Layout.AllocSize = Size2D{w, h}
	}
}

func (g *Text2D) Node2DBBox() image.Rectangle {
	return g.MyPaint.BoundingBox(g.Pos.X, g.Pos.Y, g.Pos.X+g.Layout.AllocSize.X, g.Pos.Y+g.Layout.AllocSize.Y)
}

func (g *Text2D) Render2D() {
	g.SetWinBBox(g.Node2DBBox())
	// fmt.Printf("rendering text %v\n", g.Text)
	pc := &g.MyPaint
	rs := &g.Viewport.Render
	if pc.TextLayout.Wrap {
		pc.DrawStringLines(rs, g.WrappedText, g.Pos.X, g.Pos.Y, g.Layout.AllocSize.X,
			g.Layout.AllocSize.Y)
	} else {
		pc.DrawString(rs, g.Text, g.Pos.X, g.Pos.Y, g.Layout.AllocSize.X)
	}
}

func (g *Text2D) CanReRender2D() bool {
	// todo: could optimize by checking for an opaque fill, and same bbox
	return false
}

// check for interface implementation
var _ Node2D = &Text2D{}

//////////////////////////////////////////////////////////////////////////////////
//  Utilities

type measureStringer interface {
	MeasureString(s string) (w, h float64)
}

func splitOnSpace(x string) []string {
	var result []string
	pi := 0
	ps := false
	for i, c := range x {
		s := unicode.IsSpace(c)
		if s != ps && i > 0 {
			result = append(result, x[pi:i])
			pi = i
		}
		ps = s
	}
	result = append(result, x[pi:])
	return result
}

func wordWrap(m measureStringer, s string, width float64) []string {
	var result []string
	for _, line := range strings.Split(s, "\n") {
		fields := splitOnSpace(line)
		if len(fields)%2 == 1 {
			fields = append(fields, "")
		}
		x := ""
		for i := 0; i < len(fields); i += 2 {
			w, _ := m.MeasureString(x + fields[i])
			if w > width {
				if x == "" {
					result = append(result, fields[i])
					x = ""
					continue
				} else {
					result = append(result, x)
					x = ""
				}
			}
			x += fields[i] + fields[i+1]
		}
		if x != "" {
			result = append(result, x)
		}
	}
	for i, line := range result {
		result[i] = strings.TrimSpace(line)
	}
	return result
}
