// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"errors"
	"image"
	"image/color"

	"log"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/goki/gi/units"
	"github.com/goki/prof"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/math/fixed"
)

// text.go contains all the core text rendering and formatting code -- see
// font.go for basic font-level style and management
//
// Styling, Formatting / Layout, and Rendering are each handled separately as
// three different levels in the stack -- simplifies many things to separate
// in this way, and makes the final render pass maximally efficient and
// high-performance, at the potential cost of some memory redundancy.

// RuneRender contains fully explicit data needed for rendering a single rune
// -- Face and Color can be nil after first element, in which case the last
// non-nil is used -- slightly more efficient to avoid setting all those pointers
type RuneRender struct {
	Face   font.Face   `desc:"fully-specified font rendering info, includes fully computed font size -- this is exactly what will be drawn -- no further transforms"`
	Color  color.Color `desc:"color to draw characters in"`
	RelPos Vec2D       `desc:"relative position from start of TextRender for the lower-left baseline rendering position of the font character"`
	RotRad float32     `desc:"rotation in radians for this character, relative to its lower-left baseline rendering position"`
}

// HasNil returns error if any of the key info (face, color) is nil -- only
// the first element must be non-nil
func (rr *RuneRender) HasNil() error {
	if rr.Face == nil {
		return errors.New("gi.RuneRender: Face is nil")
	}
	if rr.Color == nil {
		return errors.New("gi.RuneRender: Color is nil")
	}
	return nil
}

// Init initializes all fields to nil / zero
func (rr *RuneRender) Init() {
	rr.Face = nil
	rr.Color = nil
	rr.RelPos = Vec2DZero
	rr.RotRad = 0
}

// CurFace is convenience for updating current font face if non-nil
func (rr *RuneRender) CurFace(curFace font.Face) font.Face {
	if rr.Face != nil {
		return rr.Face
	}
	return curFace
}

// CurColor is convenience for updating current color if non-nil
func (rr *RuneRender) CurColor(curColor color.Color) color.Color {
	if rr.Color != nil {
		return rr.Color
	}
	return curColor
}

// TextRender contains fully explicit data needed for rendering a slice of
// runes, with rune and RuneRender elements in one-to-one correspondence (but
// any nil values will use prior non-nil value) -- typically a single line of
// text is so-organized, but any amount text can be represented here -- use a
// slice of TextRender elements to represent a larger block of text, which is
// used for rendering function
type TextRender struct {
	Text    []rune       `desc:"text as runes"`
	Render  []RuneRender `desc:"render info for each rune in one-to-one correspondence"`
	RelPos  Vec2D        `desc:"position for start of text relative to an absolute coordinate that is provided at the time of rendering -- individual rune RelPos are added to this plus the render-time offset to get the final position"`
	LastPos Vec2D        `desc:"rune position for right-hand-side of last rune -- for standard flat strings this is the overall length of the string -- not used for rendering but should be set to allow quick size computations"`
}

// IsValid ensures that at least some text is represented and the sizes of
// Text and Render slices are the same, and that the first render info is non-nil
func (tr *TextRender) IsValid() error {
	if len(tr.Text) == 0 {
		return errors.New("gi.TextRender: Text is empty")
	}
	if len(tr.Text) != len(tr.Render) {
		return errors.New("gi.TextRender: Render length != Text length")
	}
	return tr.Render[0].HasNil()
}

// SetString initializes TextRender to given string, with given default
// rendering parameters that are set for the first render element --
// constructs Render slice of same size as Text
func (tr *TextRender) SetString(str string, face font.Face, clr color.Color) {
	tr.Text = []rune(str)
	sz := len(tr.Text)
	if sz == 0 {
		return
	}
	tr.Render = make([]RuneRender, sz)
	// probably not worth the extra cost of initializing..
	// } else {
	// 	if len(tr.Render) > sz {
	// 		tr.Render = tr.Render[:sz]
	// 	} else {
	// 		tr.Render = append(tr.Render, [sz - len(tr.Render)]TextRender...)
	// 	}
	// }
	tr.Render[0].Face = face
	tr.Render[0].Color = clr
}

// todo: TB, RL cases..

// SetLRRunePos sets relative positions of each rune assuming a flat
// left-to-right text layout, based on font size info and additional extra
// spacing parameter (which can be negative)
func (tr *TextRender) SetLRRunePos(extraSpace float32) {
	if err := tr.IsValid(); err != nil {
		log.Println(err)
		return
	}
	sz := len(tr.Text)
	prevR := rune(-1)
	fspc := Float32ToFixed(extraSpace)
	var fpos fixed.Int26_6
	curFace := tr.Render[0].Face
	for i, r := range tr.Text {
		tr.Render[i].RelPos.X = FixedToFloat32(fpos)
		tr.Render[i].RelPos.Y = 0
		curFace := tr.Render[i].CurFace(curFace)
		if prevR >= 0 {
			fpos += curFace.Kern(prevR, r)
		}
		a, ok := curFace.GlyphAdvance(r)
		if !ok {
			// TODO: is falling back on the U+FFFD glyph the responsibility of
			// the Drawer or the Face?
			// TODO: set prevC = '\ufffd'?
			continue
		}
		fpos += a
		if i < sz-1 {
			fpos += fspc
		}
		prevR = r
	}
	tr.LastPos.X = FixedToFloat32(fpos)
	tr.LastPos.Y = 0
}

// SetLRRunePosXForm sets relative positions of each rune assuming a flat
// left-to-right text layout, based on font size info and additional extra
// spacing parameter (which can be negative), subject to given transform, and
// absolute position
func (tr *TextRender) SetLRRunePosXForm(extraSpace float32, xform XFormMatrix2D, pos Vec2D) {
	if err := tr.IsValid(); err != nil {
		log.Println(err)
		return
	}
	sz := len(tr.Text)
	prevR := rune(-1)
	fspc := Float32ToFixed(extraSpace)
	var fpos fixed.Int26_6
	curFace := tr.Render[0].Face
	for i, r := range tr.Text {
		// todo: use xform, pos -- add pos, transform back, etc..
		tr.Render[i].RelPos.X = FixedToFloat32(fpos)
		tr.Render[i].RelPos.Y = 0
		curFace := tr.Render[i].CurFace(curFace)
		if prevR >= 0 {
			fpos += curFace.Kern(prevR, r)
		}
		a, ok := curFace.GlyphAdvance(r)
		if !ok {
			// TODO: is falling back on the U+FFFD glyph the responsibility of
			// the Drawer or the Face?
			// TODO: set prevC = '\ufffd'?
			continue
		}
		fpos += a
		if i < sz-1 {
			fpos += fspc
		}
		prevR = r
	}
	tr.LastPos.X = FixedToFloat32(fpos)
	tr.LastPos.Y = 0
}

// RenderText does text rendering for a slice of TextRender text elements --
// any applicable transforms (aside from the char-specific rotation in Render)
// must be applied in advance in computing the relative positions of the
// runes, and the overall font size, etc
func RenderText(im *image.RGBA, bounds image.Rectangle, pos Vec2D, text []TextRender) {
	pr := prof.Start("RenderText")
	for _, tr := range text {
		if tr.IsValid() != nil {
			continue
		}
		curFace := tr.Render[0].Face
		curColor := tr.Render[0].Color
		tpos := pos.Add(tr.RelPos)

		fht := curFace.Metrics().Ascent // just for bounds checking

		d := &font.Drawer{
			Dst:  im,
			Src:  image.NewUniform(curColor),
			Face: curFace,
			Dot:  Float32ToFixedPoint(tpos.X, tpos.Y),
		}

		// based on Drawer.DrawString() in golang.org/x/image/font/font.go
		for i, r := range tr.Text {
			rr := &(tr.Render[i])
			rp := tpos.Add(rr.RelPos)
			if int(rp.X) > bounds.Max.X || int(rp.Y) < bounds.Min.Y {
				continue
			}
			d.Dot = Float32ToFixedPoint(rp.X, rp.Y)
			dr, mask, maskp, advance, ok := d.Face.Glyph(d.Dot, r)
			if !ok {
				continue
			}
			rx := d.Dot.X + advance
			ty := d.Dot.Y - fht
			if rx.Floor() < bounds.Min.X || ty.Ceil() > bounds.Max.Y {
				continue
			}
			sr := dr.Sub(dr.Min)

			if rr.RotRad == 0 {
				draw.Draw(d.Dst, sr, d.Src, image.ZP, draw.Over) // todo mask?
			} else {
				transformer := draw.BiLinear
				m := Rotate2D(rr.RotRad) // todo: maybe have to back the position out first..
				s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
				transformer.Transform(d.Dst, s2d, d.Src, sr, draw.Over, &draw.Options{
					SrcMask:  mask,
					SrcMaskP: maskp,
				})
			}
		}
		pr.End()
	}
}

// TextStyle contains all the style information associated with how to format
// text -- most of these are inherited
type TextStyle struct {
	Align         Align       `xml:"text-align" inherit:"true" desc:"how to align text"`
	AlignV        Align       `xml:"-" json:"-" desc:"vertical alignment of text -- copied from layout style AlignV"`
	LineHeight    float32     `xml:"line-height" inherit:"true" desc:"specified height of a line of text, in proportion to default font height, 0 = 1 = normal (note: specific values such as pixels are not supported)"`
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
func (p *TextStyle) EffLineHeight() float32 {
	if p.LineHeight == 0 {
		return 1.0
	}
	return p.LineHeight
}

// AlignFactors gets basic text alignment factors for DrawString routines --
// does not handle justified
func (p *TextStyle) AlignFactors() (ax, ay float32) {
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

// todo: all of text alignment stuff from Paint goes here.  Get

//////////////////////////////////////////////////////////////////////////////////
//  Utilities

// NextRuneAt returns the next rune starting from given index -- could be at
// that index or some point thereafter -- returns utf8.RuneError if no valid
// rune could be found -- this should be a standard function!
func NextRuneAt(str string, idx int) rune {
	r, _ := utf8.DecodeRuneInString(str[idx:])
	return r
}

// MeasureChars returns inter-character points for each rune, in float32
func MeasureChars(f font.Face, s []rune) []float32 {
	chrs := make([]float32, len(s))
	prevC := rune(-1)
	var advance fixed.Int26_6
	for i, c := range s {
		if prevC >= 0 {
			advance += f.Kern(prevC, c)
		}
		a, ok := f.GlyphAdvance(c)
		if !ok {
			// TODO: is falling back on the U+FFFD glyph the responsibility of
			// the Drawer or the Face?
			// TODO: set prevC = '\ufffd'?
			continue
		}
		advance += a
		chrs[i] = FixedToFloat32(advance)
		prevC = c
	}
	return chrs
}

type measureStringer interface {
	MeasureString(s string) (w, h float32)
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

func wordWrap(m measureStringer, s string, width float32) []string {
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
