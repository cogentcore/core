// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"github.com/golang/freetype/truetype"
	"github.com/rcoreilly/goki/ki"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	// "golang.org/x/image/font/gofont/goregular"
	"image"
	"io/ioutil"
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

// font information for painter
type PaintFont struct {
	Face   font.Face
	Height float64
}

func (p *PaintFont) Defaults() {
	// p.Face, _ = truetype.Parse(goregular.TTF)
	p.Face = basicfont.Face7x13
	p.Height = 12
}

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
func (pf *PaintFont) SetFromNode(g *Node2DBase) {
	// always check if property has been set before setting -- otherwise defaults to empty -- true = inherit props

	if nm, got := g.PropEnum("font-face"); got {
		if len(nm) != 0 {
			// todo decode font
			fmt.Printf("todo: process font face: %v\n", nm)
		}
	}
	if sz, got := g.PropNumber("font-size"); got {
		pf.Height = sz
		fmt.Printf("font height: %v\n", sz)
	}
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
// Node

// todo: lots of work likely needed on laying-out text in proper way
// https://www.w3.org/TR/SVG2/text.html#GlyphsMetrics
// todo: tspan element

// 2D Text
type Text2D struct {
	Node2DBase
	Pos  Point2D `svg:"{x,y}",desc:"position of left baseline "`
	Size Size2D  `svg:"{width,height}",desc:"size of overall text box -- width can be either entered or computed depending on wrapped"`
	Text string  `svg:"text",desc:"text string to render"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Text2D = ki.KiTypes.AddType(&Text2D{})

func (g *Text2D) GiNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Text2D) GiViewport2D() *Viewport2D {
	return nil
}

func (g *Text2D) InitNode2D(vp *Viewport2D) bool {
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *Text2D) Node2DBBox(vp *Viewport2D) image.Rectangle {
	// todo: need to update this!
	return vp.BoundingBox(g.Pos.X, g.Pos.Y, g.Pos.X+g.Size.X, g.Pos.Y+g.Size.Y)
}

func (g *Text2D) Render2D(vp *Viewport2D) bool {
	if vp.HasNoStrokeOrFill() {
		return true
	}
	vp.DrawString(g.Text, g.Pos.X, g.Pos.Y, g.Size.X)
	if vp.HasFill() {
		vp.FillPreserve()
	}
	if vp.HasStroke() {
		vp.StrokePreserve()
	}
	vp.ClearPath()
	return true
}

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

func LoadFontFace(path string, points float64) (font.Face, error) {
	fontBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		return nil, err
	}
	face := truetype.NewFace(f, &truetype.Options{
		Size: points,
		// Hinting: font.HintingFull,
	})
	return face, nil
}
