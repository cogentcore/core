// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"strings"
	"unicode"

	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki/kit"
)

// note: most of these are inherited

// all the style information associated with how to render text
type TextStyle struct {
	Align         Align       `xml:"text-align" inherit:"true" desc:"how to align text"`
	AlignV        Align       `xml:"vertical-align" alt:"vert-align,align-vert" desc:"vertical alignment of text -- copied from layout style AlignV"`
	LineHeight    float64     `xml:"line-height" inherit:"true" desc:"specified height of a line of text, in proportion to default font height, 0 = 1 = normal (note: specific values such as pixels are not supported)"`
	LetterSpacing units.Value `xml:"letter-spacing" desc:"spacing between characters and lines"`
	Indent        units.Value `xml:"text-indent" inherit:"true" desc:"how much to indent the first line in a paragraph"`
	TabSize       units.Value `xml:"tab-size" inherit:"true" desc:"tab size"`
	WordSpacing   units.Value `xml:"word-spacing" inherit:"true" desc:"extra space to add between words"`
	WordWrap      bool        `xml:"word-wrap" inherit:"true" desc:"wrap text within a given size"`
	// todo:
	// page-break options
	// text-decoration-line -- underline, overline, line-through, -style, -color inherit
	// text-justify  inherit:"true" -- how to justify text
	// text-overflow -- clip, ellipsis, string..
	// text-shadow  inherit:"true"
	// text-transform --  inherit:"true" uppercase, lowercase, capitalize
	// user-select -- can user select text?
	// white-space -- what to do with white-space  inherit:"true"
	// word-break  inherit:"true"
}

func (p *TextStyle) Defaults() {
	p.WordWrap = false
	p.Align = AlignLeft
	p.AlignV = AlignBaseline
}

// any updates after generic xml-tag property setting?
func (p *TextStyle) SetStylePost() {
}

// effective line height (taking into account 0 value)
func (p *TextStyle) EffLineHeight() float64 {
	if p.LineHeight == 0 {
		return 1.0
	}
	return p.LineHeight
}

// get basic text alignment factors for DrawString routines -- does not handle justified
func (p *TextStyle) AlignFactors() (ax, ay float64) {
	ax = 0.0
	ay = 0.0
	hal := p.Align
	switch {
	case IsAlignMiddle(hal):
		ax = 0.5 // todo: determine if font is horiz or vert..
	case IsAlignEnd(hal):
		ax = 1.0
	}
	val := p.AlignV
	switch {
	case val == AlignSub:
		ay = -0.1 // todo: fixme -- need actual font metrics
	case val == AlignSuper:
		ay = 0.8 // todo: fixme
	case IsAlignStart(val):
		ay = 0.9 // todo: need to find out actual baseline
	case IsAlignMiddle(val):
		ay = 0.45 // todo: determine if font is horiz or vert..
	case IsAlignEnd(val):
		ay = -0.1 // todo: need actual baseline
	}
	return
}

////////////////////////////////////////////////////////////////////////////////////////
// Text2D Node

// todo: lots of work likely needed on laying-out text in proper way
// https://www.w3.org/TR/SVG2/text.html#GlyphsMetrics
// todo: tspan element

// 2D Text
type Text2D struct {
	Node2DBase
	Pos         Vec2D    `xml:"{x,y}" desc:"position of the left, baseline of the text"`
	Width       float64  `xml:"width" desc:"width of text to render if using word-wrapping"`
	Text        string   `xml:"text" desc:"text string to render"`
	WrappedText []string `json:"-","desc:word-wrapped version of the string"`
}

var KiT_Text2D = kit.Types.AddType(&Text2D{}, nil)

func (g *Text2D) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Text2D) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Text2D) AsLayout2D() *Layout {
	return nil
}

func (g *Text2D) Init2D() {
	g.Init2DBase()
	g.LayData.Defaults()
}

func (g *Text2D) Style2D() {
	g.Style2DSVG(nil)
}

func (g *Text2D) Size2D() {
	g.InitLayout2D()
	pc := &g.Paint
	var w, h float64
	// pre-wrap the text
	if pc.TextStyle.WordWrap {
		g.WrappedText, h = pc.MeasureStringWrapped(g.Text, g.Width, pc.TextStyle.EffLineHeight())
	} else {
		w, h = pc.MeasureString(g.Text)
	}
	g.LayData.AllocSize = Vec2D{w, h}
}

func (g *Text2D) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, false) // no style
	g.Layout2DChildren()
}

func (g *Text2D) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	return g.Paint.BoundingBox(rs, g.Pos.X, g.Pos.Y, g.Pos.X+g.LayData.AllocSize.X, g.Pos.Y+g.LayData.AllocSize.Y)
}

func (g *Text2D) ComputeBBox2D(parBBox image.Rectangle) {
	g.ComputeBBox2DBase(parBBox)
}

func (g *Text2D) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox
}

func (g *Text2D) Move2D(delta Vec2D, parBBox image.Rectangle) {
	g.Move2DBase(delta, parBBox) // moves parts
	g.Move2DChildren(delta)
}

func (g *Text2D) Render2D() {
	if g.PushBounds() {
		pc := &g.Paint
		rs := &g.Viewport.Render
		// pc.SetUnitContext(g.Viewport, Vec2DZero) // todo: not sure about el
		// fmt.Printf("rendering text %v\n", g.Text)
		if pc.TextStyle.WordWrap {
			pc.DrawStringLines(rs, g.WrappedText, g.Pos.X, g.Pos.Y, g.LayData.AllocSize.X,
				g.LayData.AllocSize.Y)
		} else {
			pc.DrawString(rs, g.Text, g.Pos.X, g.Pos.Y, g.LayData.AllocSize.X)
		}
		g.Render2DChildren()
		g.PopBounds()
	}
}

func (g *Text2D) ReRender2D() (node Node2D, layout bool) {
	svg := g.ParentSVG()
	if svg != nil {
		node = svg
	} else {
		node = g.This.(Node2D) // no other option..
	}
	layout = false
	return
}

func (g *Text2D) FocusChanged2D(gotFocus bool) {
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
