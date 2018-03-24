// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"image"
	// "log"
	"strings"
	"unicode"
)

type TextAlign int

const (
	TextAlignLeft TextAlign = iota
	TextAlignCenter
	TextAlignRight
	TextAlignJustify
	TextAlignN
)

//go:generate stringer -type=TextAlign

var KiT_TextAlign = ki.Enums.AddEnumAltLower(TextAlignLeft, false, nil, "TextAlign", int64(TextAlignN))

// note: most of these are inherited

// all the style information associated with how to render text
type TextStyle struct {
	Align         TextAlign   `xml:"text-align",inherit:"true",desc:"how to align text"`
	LineHeight    float64     `xml:"line-height",inherit:"true",desc:"specified height of a line of text, in proportion to default font height, 0 = 1 = normal (note: specific values such as pixels are not supported)"`
	LetterSpacing units.Value `xml:"letter-spacing",desc:"spacing between characters and lines"`
	Indent        units.Value `xml:"text-indent",inherit:"true",desc:"how much to indent the first line in a paragraph"`
	TabSize       units.Value `xml:"tab-size",inherit:"true",desc:"tab size"`
	WordSpacing   units.Value `xml:"word-spacing",inherit:"true",desc:"extra space to add between words"`
	WordWrap      bool        `xml:"word-wrap",inherit:"true",desc:"wrap text within a given size"`
	// todo:
	// page-break options
	// text-decoration-line -- underline, overline, line-through, -style, -color inherit
	// text-justify ,inherit:"true" -- how to justify text
	// text-overflow -- clip, ellipsis, string..
	// text-shadow ,inherit:"true"
	// text-transform -- ,inherit:"true" uppercase, lowercase, capitalize
	// user-select -- can user select text?
	// white-space -- what to do with white-space ,inherit:"true"
	// word-break ,inherit:"true"
}

func (p *TextStyle) Defaults() {
	p.WordWrap = false
	p.Align = TextAlignLeft
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

////////////////////////////////////////////////////////////////////////////////////////
// Text2D Node

// todo: lots of work likely needed on laying-out text in proper way
// https://www.w3.org/TR/SVG2/text.html#GlyphsMetrics
// todo: tspan element

// 2D Text
type Text2D struct {
	Node2DBase
	Pos         Vec2D    `xml:"{x,y}",desc:"position of the left, baseline of the text"`
	Width       float64  `xml:"width",desc:"width of text to render if using word-wrapping"`
	Text        string   `xml:"text",desc:"text string to render"`
	WrappedText []string `json:"-","desc:word-wrapped version of the string"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Text2D = ki.Types.AddType(&Text2D{}, nil)

func (g *Text2D) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Text2D) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Text2D) AsLayout2D() *Layout {
	return nil
}

func (g *Text2D) InitNode2D() {
	g.InitNode2DBase()
	g.LayData.Defaults()
}

func (g *Text2D) Style2D() {
	g.Style2DSVG()
}

func (g *Text2D) Layout2D(iter int) {
	if iter == 0 {
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
	} else {
		g.GeomFromLayout()
	}
}

func (g *Text2D) Node2DBBox() image.Rectangle {
	return g.Paint.BoundingBox(g.Pos.X, g.Pos.Y, g.Pos.X+g.LayData.AllocSize.X, g.Pos.Y+g.LayData.AllocSize.Y)
}

func (g *Text2D) Render2D() {
	pc := &g.Paint
	rs := &g.Viewport.Render
	pc.SetUnitContext(rs, 0) // todo: not sure about el
	g.SetWinBBox(g.Node2DBBox())
	// fmt.Printf("rendering text %v\n", g.Text)
	if pc.TextStyle.WordWrap {
		pc.DrawStringLines(rs, g.WrappedText, g.Pos.X, g.Pos.Y, g.LayData.AllocSize.X,
			g.LayData.AllocSize.Y)
	} else {
		pc.DrawString(rs, g.Text, g.Pos.X, g.Pos.Y, g.LayData.AllocSize.X)
	}
}

func (g *Text2D) CanReRender2D() bool {
	// todo: could optimize by checking for an opaque fill, and same bbox
	return false
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
