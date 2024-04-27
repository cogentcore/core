// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"fmt"
	"image"

	"cogentcore.org/core/colors"
)

// SVGStart returns the start of an SVG based on the current context state
func (pc *Context) SVGStart() string {
	sz := pc.Image.Bounds().Size()
	return fmt.Sprintf(`<svg width="%dpx" height="%dpx">\n`, sz.X, sz.Y)
}

// SVGEnd returns the end of an SVG based on the current context state
func (pc *Context) SVGEnd() string {
	return "</svg>"
}

// SVGPath generates an SVG path representation of the current Path
func (pc *Context) SVGPath() string {
	style := pc.SVGStrokeStyle() + pc.SVGFillStyle()
	return `<path style="` + style + `" d="` + pc.Path.ToSVGPath() + `"/>\n`
}

// SVGStrokeStyle returns the style string for current Stroke
func (pc *Context) SVGStrokeStyle() string {
	if pc.StrokeStyle.Color == nil {
		return "stroke:none;"
	}
	s := "stroke-width:" + fmt.Sprintf("%g", pc.StrokeWidth()) + ";"
	switch im := pc.StrokeStyle.Color.(type) {
	case *image.Uniform:
		s += "stroke:" + colors.AsHex(colors.AsRGBA(im)) + ";"
	}
	// todo: dashes, gradients
	return s
}

// SVGFillStyle returns the style string for current Fill
func (pc *Context) SVGFillStyle() string {
	if pc.FillStyle.Color == nil {
		return "fill:none;"
	}
	s := ""
	switch im := pc.FillStyle.Color.(type) {
	case *image.Uniform:
		s += "fill:" + colors.AsHex(colors.AsRGBA(im)) + ";"
	}
	// todo: gradients etc
	return s
}
