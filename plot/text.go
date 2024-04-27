// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
)

// DefaultFontFamily specifies a default font for plotting.
// if not set, the standard Cogent Core default font is used.
var DefaultFontFamily = ""

// TextStyle specifies styling parameters for Text elements
type TextStyle struct {
	styles.FontRender

	// how to align text along the relevant dimension for the text element
	Align styles.Aligns

	// rotation of the text
	Rotation float32
}

func (ts *TextStyle) Defaults() {
	ts.FontRender.Defaults()
	if DefaultFontFamily != "" {
		ts.FontRender.Family = DefaultFontFamily
	}
}

// Text specifies a single text element in a plot
type Text struct {

	// text string, which can use HTML formatting
	Text string

	// styling for this text element
	Style TextStyle

	// paintText is the [paint.Text] for the text.
	paintText paint.Text
}

func (tx *Text) Defaults() {
	tx.Style.Defaults()
}

// Config is called during the layout of the plot, prior to drawing
func (tx *Text) Config(pt *Plot) {
	fs := &tx.Style.FontRender
	txln := float32(len(tx.Text))
	fht := float32(16)
	hsz := float32(12) * txln
	if fs.Face != nil {
		fht = fs.Face.Metrics.Height
		hsz = 0.75 * fht * txln
	}
	txs := &pt.StdTextStyle
	txs.OrientationHoriz = tx.Style.Rotation
	txs.Align = tx.Style.Align

	tx.paintText.SetHTML(tx.Text, fs, txs, &pt.UnitContext, nil)
	tx.paintText.Layout(txs, fs, &pt.UnitContext, math32.Vector2{X: hsz, Y: fht})
}

// Draw renders the text at given upper left position
func (tx *Text) Draw(pt *Plot, pos math32.Vector2) {
	tx.paintText.Render(&pt.Paint, pos)
}
