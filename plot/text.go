// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// DefaultFontFamily specifies a default font for plotting.
// if not set, the standard Cogent Core default font is used.
var DefaultFontFamily = ""

// TextStyle specifies styling parameters for Text elements.
type TextStyle struct { //types:add -setters
	// Size of font to render. Default is 16dp
	Size units.Value

	// Family name for font (inherited): ordered list of comma-separated names
	// from more general to more specific to use. Use split on, to parse.
	Family string

	// Color of text.
	Color image.Image

	// Align specifies how to align text along the relevant
	// dimension for the text element.
	Align styles.Aligns

	// Padding is used in a case-dependent manner to add
	// space around text elements.
	Padding units.Value

	// Rotation of the text, in degrees.
	Rotation float32

	// Offset is added directly to the final label location.
	Offset units.XY
}

func (ts *TextStyle) Defaults() {
	ts.Size.Dp(16)
	ts.Color = colors.Scheme.OnSurface
	ts.Align = styles.Center
	if DefaultFontFamily != "" {
		ts.Family = DefaultFontFamily
	}
}

// Text specifies a single text element in a plot
type Text struct {

	// text string, which can use HTML formatting
	Text string

	// styling for this text element
	Style TextStyle

	// font has the full font rendering styles.
	font styles.FontRender

	// PaintText is the [paint.Text] for the text.
	PaintText paint.Text
}

func (tx *Text) Defaults() {
	tx.Style.Defaults()
}

// config is called during the layout of the plot, prior to drawing
func (tx *Text) Config(pt *Plot) {
	uc := &pt.Paint.UnitContext
	fs := &tx.font
	fs.Size = tx.Style.Size
	fs.Family = tx.Style.Family
	fs.Color = tx.Style.Color
	if math32.Abs(tx.Style.Rotation) > 10 {
		tx.Style.Align = styles.End
	}
	fs.ToDots(uc)
	tx.Style.Padding.ToDots(uc)
	txln := float32(len(tx.Text))
	fht := fs.Size.Dots
	hsz := float32(12) * txln
	txs := &pt.StandardTextStyle
	tx.PaintText.SetHTML(tx.Text, fs, txs, uc, nil)
	tx.PaintText.Layout(txs, fs, uc, math32.Vector2{X: hsz, Y: fht})
	if tx.Style.Rotation != 0 {
		rotx := math32.Rotate2D(math32.DegToRad(tx.Style.Rotation))
		tx.PaintText.Transform(rotx, fs, uc)
	}
}

func (tx *Text) openFont(pt *Plot) {
	if tx.font.Face == nil {
		paint.OpenFont(&tx.font, &pt.Paint.UnitContext) // calls SetUnContext after updating metrics
	}
}

func (tx *Text) ToDots(uc *units.Context) {
	tx.font.ToDots(uc)
	tx.Style.Padding.ToDots(uc)
}

// PosX returns the starting position for a horizontally-aligned text element,
// based on given width.  Text must have been config'd already.
func (tx *Text) PosX(width float32) math32.Vector2 {
	pos := math32.Vector2{}
	pos.X = styles.AlignFactor(tx.Style.Align) * width
	switch tx.Style.Align {
	case styles.Center:
		pos.X -= 0.5 * tx.PaintText.BBox.Size().X
	case styles.End:
		pos.X -= tx.PaintText.BBox.Size().X
	}
	if math32.Abs(tx.Style.Rotation) > 10 {
		pos.Y += 0.5 * tx.PaintText.BBox.Size().Y
	}
	return pos
}

// PosY returns the starting position for a vertically-rotated text element,
// based on given height.  Text must have been config'd already.
func (tx *Text) PosY(height float32) math32.Vector2 {
	pos := math32.Vector2{}
	pos.Y = styles.AlignFactor(tx.Style.Align) * height
	switch tx.Style.Align {
	case styles.Center:
		pos.Y -= 0.5 * tx.PaintText.BBox.Size().Y
	case styles.End:
		pos.Y -= tx.PaintText.BBox.Size().Y
	}
	return pos
}

// Draw renders the text at given upper left position
func (tx *Text) Draw(pt *Plot, pos math32.Vector2) {
	tx.PaintText.Render(pt.Paint, pos)
}
