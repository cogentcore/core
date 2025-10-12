// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package pdf

import (
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"reflect"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
)

// SetFill sets the fill style values where different from current.
func (w *pdfPage) SetFill(fill *styles.Fill) {
	csty := w.style()
	if csty.Fill.Color != fill.Color || csty.Fill.Opacity != fill.Opacity {
		w.SetFillColor(fill)
	}
	csty.Fill = *fill
}

// SetAlpha sets the transparency value.
func (w *pdfPage) SetAlpha(alpha float32) {
	gs := w.getOpacityGS(alpha)
	fmt.Fprintf(w, " /%v gs", gs)
}

func alphaNorm(c uint8, a float32) dec {
	if a == 0 {
		return dec(0)
	}
	return dec(float32(c) / 255.0 / a)
}

func alphaNormColor(c color.RGBA, a float32) [3]dec {
	var v [3]dec
	v[0] = alphaNorm(c.R, a)
	v[1] = alphaNorm(c.G, a)
	v[2] = alphaNorm(c.B, a)
	return v
}

// SetFillColor sets the filling color (image).
func (w *pdfPage) SetFillColor(fill *styles.Fill) {
	switch x := fill.Color.(type) {
	// todo: image
	case *gradient.Linear:
		fmt.Fprintf(w, " /Pattern cs /%v scn", w.getPattern(x))
	case *gradient.Radial:
		fmt.Fprintf(w, " /Pattern cs /%v scn", w.getPattern(x))
	case *image.Uniform:
		var clr color.RGBA
		if x != nil {
			clr = colors.ApplyOpacity(colors.AsRGBA(x), fill.Opacity)
		}
		a := float32(clr.A) / 255.0
		if a == 0 {
			fmt.Fprintf(w, " 0 g")
		} else if clr.R == clr.G && clr.R == clr.B {
			fmt.Fprintf(w, " %v g", alphaNorm(clr.R, a))
		} else {
			v := alphaNormColor(clr, a)
			fmt.Fprintf(w, " %v %v %v rg", v[0], v[1], v[2])
		}
		w.SetAlpha(a)
	}
}

// SetStroke sets the stroke style values where different from current.
func (w *pdfPage) SetStroke(stroke *styles.Stroke) {
	csty := w.style()
	if csty.Stroke.Color != stroke.Color || csty.Stroke.Opacity != stroke.Opacity {
		w.SetStrokeColor(stroke)
	}
	if csty.Stroke.Width.Dots != stroke.Width.Dots {
		w.SetStrokeWidth(stroke.Width.Dots)
	}
	if csty.Stroke.Cap != stroke.Cap {
		w.SetStrokeCap(stroke.Cap)
	}
	if csty.Stroke.Join != stroke.Join || (csty.Stroke.Join == ppath.JoinMiter && csty.Stroke.MiterLimit != stroke.MiterLimit) {
		w.SetStrokeJoin(stroke.Join, stroke.MiterLimit)
	}
	if len(stroke.Dashes) > 0 { // always do
		w.SetDashes(stroke.DashOffset, stroke.Dashes)
	} else {
		if len(csty.Stroke.Dashes) > 0 {
			w.SetDashes(0, nil)
		}
	}
	csty.Stroke = *stroke
}

// SetStrokeColor sets the stroking color (image).
func (w *pdfPage) SetStrokeColor(stroke *styles.Stroke) {
	switch x := stroke.Color.(type) {
	case *gradient.Linear:
	case *gradient.Radial:
		// TODO: should we unset cs?
		// fmt.Fprintf(w, " /Pattern cs /%v scn", w.getPattern(stroke.Gradient))
	case *image.Uniform:
		clr := colors.ApplyOpacity(colors.AsRGBA(x), stroke.Opacity)
		a := float32(clr.A) / 255.0
		if clr.R == clr.G && clr.R == clr.B {
			fmt.Fprintf(w, " %v G", alphaNorm(clr.R, a))
		} else {
			v := alphaNormColor(clr, a)
			fmt.Fprintf(w, " %v %v %v RG", v[0], v[1], v[2])
		}
		w.SetAlpha(a)
	}
}

// SetStrokeWidth sets the stroke width.
func (w *pdfPage) SetStrokeWidth(lineWidth float32) {
	fmt.Fprintf(w, " %v w", dec(lineWidth))
}

// SetStrokeCap sets the stroke cap type.
func (w *pdfPage) SetStrokeCap(capper ppath.Caps) {
	var lineCap int
	switch capper {
	case ppath.CapButt:
		lineCap = 0
	case ppath.CapRound:
		lineCap = 1
	case ppath.CapSquare:
		lineCap = 2
	default:
		slog.Error("pdfWriter", "StrokeCap not supported", capper)
	}
	fmt.Fprintf(w, " %d J", lineCap)
}

// SetStrokeJoin sets the stroke join type.
func (w *pdfPage) SetStrokeJoin(joiner ppath.Joins, miterLimit float32) {
	var lineJoin int
	switch joiner {
	case ppath.JoinBevel:
		lineJoin = 2
	case ppath.JoinRound:
		lineJoin = 1
	case ppath.JoinMiter:
		lineJoin = 0
	default:
		slog.Error("pdfWriter", "StrokeJoin not supported", joiner)
	}
	fmt.Fprintf(w, " %d j", lineJoin)
	if lineJoin == 0 {
		fmt.Fprintf(w, " %v M", dec(miterLimit))
	}
}

// SetDashes sets the dash phase and array.
func (w *pdfPage) SetDashes(dashPhase float32, dashArray []float32) {
	if len(dashArray)%2 == 1 {
		dashArray = append(dashArray, dashArray...)
	}

	// PDF can't handle negative dash phases
	if dashPhase < 0.0 {
		totalLength := float32(0.0)
		for _, dash := range dashArray {
			totalLength += dash
		}
		for dashPhase < 0.0 {
			dashPhase += totalLength
		}
	}

	dashes := append(dashArray, dashPhase)
	if len(dashes) == 1 {
		fmt.Fprintf(w, " [] 0 d")
		dashes[0] = 0.0
	} else {
		fmt.Fprintf(w, " [%v", dec(dashes[0]))
		for _, dash := range dashes[1 : len(dashes)-1] {
			fmt.Fprintf(w, " %v", dec(dash))
		}
		fmt.Fprintf(w, "] %v d", dec(dashes[len(dashes)-1]))
	}
}

func (w *pdfPage) getOpacityGS(a float32) pdfName {
	if name, ok := w.graphicsStates[a]; ok {
		return name
	}
	name := pdfName(fmt.Sprintf("A%d", len(w.graphicsStates)))
	w.graphicsStates[a] = name

	if _, ok := w.resources["ExtGState"]; !ok {
		w.resources["ExtGState"] = pdfDict{}
	}
	w.resources["ExtGState"].(pdfDict)[name] = pdfDict{
		"CA": a,
		"ca": a,
	}
	return name
}

func (w *pdfPage) getPattern(gr gradient.Gradient) pdfName {
	// TODO: support patterns/gradients with alpha channel
	shading := pdfDict{
		"ColorSpace": pdfName("DeviceRGB"),
	}
	switch g := gr.(type) {
	case *gradient.Linear:
		shading["ShadingType"] = 2
		shading["Coords"] = pdfArray{g.Start.X, g.Start.Y, g.End.X, g.End.Y}
		shading["Function"] = patternStopsFunction(g.Stops)
		shading["Extend"] = pdfArray{true, true}
	case *gradient.Radial:
		shading["ShadingType"] = 3
		shading["Coords"] = pdfArray{g.Center.X, g.Center.Y, g.Radius.X, g.Focal.X, g.Focal.Y, g.Radius.Y}
		shading["Function"] = patternStopsFunction(g.Stops)
		shading["Extend"] = pdfArray{true, true}
	}
	pattern := pdfDict{
		"Type":        pdfName("Pattern"),
		"PatternType": 2,
		"Shading":     shading,
	}

	if _, ok := w.resources["Pattern"]; !ok {
		w.resources["Pattern"] = pdfDict{}
	}
	for name, pat := range w.resources["Pattern"].(pdfDict) {
		if reflect.DeepEqual(pat, pattern) {
			return name
		}
	}
	name := pdfName(fmt.Sprintf("P%d", len(w.resources["Pattern"].(pdfDict))))
	w.resources["Pattern"].(pdfDict)[name] = pattern
	return name
}

func patternStopsFunction(stops []gradient.Stop) pdfDict {
	if len(stops) < 2 {
		return pdfDict{}
	}

	fs := []pdfDict{}
	encode := pdfArray{}
	bounds := pdfArray{}
	if !ppath.Equal(stops[0].Pos, 0.0) {
		fs = append(fs, patternStopFunction(stops[0], stops[0]))
		encode = append(encode, 0, 1)
		bounds = append(bounds, stops[0].Pos)
	}
	for i := 0; i < len(stops)-1; i++ {
		fs = append(fs, patternStopFunction(stops[i], stops[i+1]))
		encode = append(encode, 0, 1)
		if i != 0 {
			bounds = append(bounds, stops[1].Pos)
		}
	}
	if !ppath.Equal(stops[len(stops)-1].Pos, 1.0) {
		fs = append(fs, patternStopFunction(stops[len(stops)-1], stops[len(stops)-1]))
		encode = append(encode, 0, 1)
	}
	if len(fs) == 1 {
		return fs[0]
	}
	return pdfDict{
		"FunctionType": 3,
		"Domain":       pdfArray{0, 1},
		"Encode":       encode,
		"Bounds":       bounds,
		"Functions":    fs,
	}
}

func patternStopFunction(s0, s1 gradient.Stop) pdfDict {
	a0 := float32(s0.Color.A) / 255.0
	a1 := float32(s1.Color.A) / 255.0
	c0 := alphaNormColor(s0.Color, a0)
	c1 := alphaNormColor(s1.Color, a1)
	return pdfDict{
		"FunctionType": 2,
		"Domain":       pdfArray{0, 1},
		"N":            1,
		"C0":           pdfArray{c0[0], c0[1], c0[2]},
		"C1":           pdfArray{c1[0], c1[1], c1[2]},
	}
}
