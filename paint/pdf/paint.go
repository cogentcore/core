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
	"maps"
	"reflect"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
)

// Path renders a path to the canvas using a style and an
// individual matrix (needed for fill)
func (r *PDF) Path(path ppath.Path, style *styles.Paint, tr math32.Matrix2) {
	// TODO: PDFs don't support the arcs joiner, miter joiner (not clipped),
	// or miter joiner (clipped) with non-bevel fallback, which could be
	// drawn by filling the outline of the stroke (see stroke.Stroke).
	scale := math32.Sqrt(math32.Abs(tr.Det()))
	stk := style.Stroke
	stk.Width.Dots *= scale
	stk.DashOffset, stk.Dashes = ppath.ScaleDash(scale, stk.DashOffset, stk.Dashes)

	// PDFs don't support connecting first and last dashes if path is closed,
	// so we move the start of the path if this is the case
	// TODO: closing dashes
	//if style.DashesClose {
	//	strokeUnsupported = true
	//}

	closed := false
	trpath := path.Clone().Transform(tr)
	data := trpath.ToPDF()
	if 1 < len(data) && data[len(data)-1] == 'h' {
		data = data[:len(data)-2]
		closed = true
	}

	// the bounds of the path, for a gradient fill or stroke
	var bounds math32.Box2
	_, fillGrad := style.Fill.Color.(gradient.Gradient)
	_, strokeGrad := style.Stroke.Color.(gradient.Gradient)
	if (style.HasFill() && fillGrad) || (style.HasStroke() && strokeGrad) {
		fbox := trpath.FastBounds()
		cum := r.Cumulative()
		bounds = fbox.MulMatrix2(cum)
	}

	if style.HasFill() && !style.HasStroke() {
		r.w.SetFill(&style.Fill, bounds, tr)
		r.w.Write([]byte(" "))
		r.w.Write([]byte(data))
		r.w.Write([]byte(" f"))
		if style.Fill.Rule == ppath.EvenOdd {
			r.w.Write([]byte("*"))
		}
		r.w.endMask()
	} else if !style.HasFill() && style.HasStroke() {
		r.w.SetStroke(&stk, bounds, tr)
		r.w.Write([]byte(" "))
		r.w.Write([]byte(data))
		if closed {
			r.w.Write([]byte(" s"))
		} else {
			r.w.Write([]byte(" S"))
		}
		r.w.endMask()
	} else if style.HasFill() && style.HasStroke() {
		r.w.SetFill(&style.Fill, bounds, tr)
		r.w.Write([]byte(" "))
		r.w.Write([]byte(data))
		r.w.Write([]byte(" f"))
		if style.Fill.Rule == ppath.EvenOdd {
			r.w.Write([]byte("*"))
		}
		r.w.endMask()

		r.w.SetStroke(&stk, bounds, tr)
		r.w.Write([]byte(" "))
		r.w.Write([]byte(data))
		if closed {
			r.w.Write([]byte(" s"))
		} else {
			r.w.Write([]byte(" S"))
		}
		r.w.endMask()
	}
}

// SetFill sets the fill style values where different from current.
// The bounds and matrix are required for gradient fills: pass identity
// if not available.
func (w *pdfPage) SetFill(fill *styles.Fill, bounds math32.Box2, m math32.Matrix2) {
	csty := w.style()
	_, grad := fill.Color.(gradient.Gradient)
	// a gradient is set again for the bounds of each path
	if grad || csty.Fill.Color != fill.Color || csty.Fill.Opacity != fill.Opacity {
		w.SetFillColor(fill, bounds, m)
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
func (w *pdfPage) SetFillColor(fill *styles.Fill, bounds math32.Box2, m math32.Matrix2) {
	switch x := fill.Color.(type) {
	// todo: image
	case gradient.Gradient:
		w.setGradient(x, fill.Opacity, bounds, m, "cs", "scn")
	case *image.Uniform:
		w.endMask()
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
// The bounds and matrix are required for gradient strokes: pass identity
// if not available.
func (w *pdfPage) SetStroke(stroke *styles.Stroke, bounds math32.Box2, m math32.Matrix2) {
	csty := w.style()
	_, grad := stroke.Color.(gradient.Gradient)
	// a gradient is set again for the bounds of each path
	if grad || csty.Stroke.Color != stroke.Color || csty.Stroke.Opacity != stroke.Opacity {
		w.SetStrokeColor(stroke, bounds, m)
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

// SetStrokeColor sets the stroking color (image). The bounds and matrix
// are required for gradient strokes.
func (w *pdfPage) SetStrokeColor(stroke *styles.Stroke, bounds math32.Box2, m math32.Matrix2) {
	switch x := stroke.Color.(type) {
	case gradient.Gradient:
		w.setGradient(x, stroke.Opacity, bounds, m, "CS", "SCN")
	case *image.Uniform:
		w.endMask()
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

// setGradient sets the given gradient as the fill (cs, scn) or stroke
// (CS, SCN) color, with the given opacity. A gradient with transparent
// stops also sets a soft mask of its alphas, which [pdfPage.endMask]
// removes after the path is painted.
func (w *pdfPage) setGradient(gr gradient.Gradient, opacity float32, bounds math32.Box2, m math32.Matrix2, cs, scn string) {
	w.endMask()
	pat, mask := w.gradientPattern(gr, opacity, bounds, m)
	w.SetAlpha(1) // the transparency is in the mask
	if mask != "" {
		fmt.Fprintf(w, " /%v gs", mask)
		w.maskSet = true
	}
	fmt.Fprintf(w, " /Pattern %s /%v %s", cs, pat, scn)
}

// endMask removes the soft mask of a transparent gradient, if set.
func (w *pdfPage) endMask() {
	if !w.maskSet {
		return
	}
	fmt.Fprintf(w, " /%v gs", w.extGState("MN", pdfDict{"SMask": pdfName("None")}))
	w.maskSet = false
}

// extGState returns the name of an ExtGState with the given values,
// with names starting with the given prefix.
func (w *pdfPage) extGState(prefix string, gs pdfDict) pdfName {
	if _, ok := w.resources["ExtGState"]; !ok {
		w.resources["ExtGState"] = pdfDict{}
	}
	egs := w.resources["ExtGState"].(pdfDict)
	for name, g := range egs {
		if reflect.DeepEqual(g, gs) {
			return name
		}
	}
	name := pdfName(fmt.Sprintf("%s%d", prefix, len(egs)))
	egs[name] = gs
	return name
}

// gradientPattern returns the name of the pattern of the given gradient,
// and of the ExtGState of its soft mask if any of its stops are
// transparent (with the given opacity), or "".
func (w *pdfPage) gradientPattern(gr gradient.Gradient, opacity float32, bounds math32.Box2, m math32.Matrix2) (pdfName, pdfName) {
	cum := w.Cumulative().Mul(m)
	gr.Update(opacity, bounds, cum)
	base := gr.AsBase()
	// the geometry of the shading, the same for its colors and alphas
	geom := pdfDict{"Extend": pdfArray{true, true}}
	switch g := gr.(type) {
	case *gradient.Linear:
		s, e := g.TransformedAxis()
		geom["ShadingType"] = 2
		geom["Coords"] = pdfArray{s.X, s.Y, e.X, e.Y}
	case *gradient.Radial:
		c, f, rad := g.TransformedCoords()
		r := max(math32.Abs(rad.X), math32.Abs(rad.Y))
		geom["ShadingType"] = 3
		geom["Coords"] = pdfArray{f.X, f.Y, 0, c.X, c.Y, r}
	}
	shading := pdfDict{"ColorSpace": pdfName("DeviceRGB")}
	maps.Copy(shading, geom)
	shading["Function"] = patternStopsFunction(base.Stops, stopColor)
	pattern := pdfDict{
		"Type":        pdfName("Pattern"),
		"PatternType": 2,
		"Shading":     shading,
	}
	if _, ok := w.resources["Pattern"]; !ok {
		w.resources["Pattern"] = pdfDict{}
	}
	pats := w.resources["Pattern"].(pdfDict)
	var name pdfName
	for pn, pat := range pats {
		if reflect.DeepEqual(pat, pattern) {
			name = pn
			break
		}
	}
	if name == "" {
		name = pdfName(fmt.Sprintf("P%d", len(pats)))
		pats[name] = pattern
	}

	opaque := true
	for i := range base.Stops {
		if stopAlpha(&base.Stops[i], opacity) < 1 {
			opaque = false
			break
		}
	}
	if opaque {
		return name, ""
	}
	// the mask is a luminosity transparency group of a gray shading of the
	// alphas, in the default space of the page, as the pattern is.
	alphas := pdfDict{"ColorSpace": pdfName("DeviceGray")}
	maps.Copy(alphas, geom)
	alphas["Function"] = patternStopsFunction(base.Stops, func(st *gradient.Stop) pdfArray {
		return pdfArray{dec(stopAlpha(st, opacity))}
	})
	inv := w.Cumulative().Inverse()
	form := w.pdf.writeObject(pdfStream{
		dict: pdfDict{
			"Type":    pdfName("XObject"),
			"Subtype": pdfName("Form"),
			"BBox":    pdfArray{0, 0, w.width, w.height},
			"Matrix":  pdfArray{dec(inv.XX), dec(inv.XY), dec(inv.YX), dec(inv.YY), dec(inv.X0), dec(inv.Y0)}, // as mat2
			"Group": pdfDict{
				"Type": pdfName("Group"),
				"S":    pdfName("Transparency"),
				"CS":   pdfName("DeviceGray"),
			},
			"Resources": pdfDict{"Shading": pdfDict{"Sh0": alphas}},
		},
		stream: []byte("/Sh0 sh"),
	})
	mask := w.extGState("M", pdfDict{
		"SMask": pdfDict{
			"Type": pdfName("Mask"),
			"S":    pdfName("Luminosity"),
			"G":    form,
		},
	})
	return name, mask
}

// stopAlpha returns the alpha of the given stop, with the given opacity.
func stopAlpha(st *gradient.Stop, opacity float32) float32 {
	return float32(st.Color.A) / 255 * st.Opacity * opacity
}

// stopColor returns the (not premultiplied) RGB color of the given stop.
func stopColor(st *gradient.Stop) pdfArray {
	c := alphaNormColor(st.Color, float32(st.Color.A)/255)
	return pdfArray{c[0], c[1], c[2]}
}

// patternStopsFunction returns the function of the values of the given
// stops (as their colors or alphas), stitching the functions between
// each pair of stops.
func patternStopsFunction(stops []gradient.Stop, value func(st *gradient.Stop) pdfArray) pdfDict {
	n := len(stops)
	if n == 0 {
		return pdfDict{}
	}
	if n == 1 {
		return patternStopFunction(value(&stops[0]), value(&stops[0]))
	}

	fs := pdfArray{}
	encode := pdfArray{}
	bounds := pdfArray{}
	if !ppath.Equal(stops[0].Pos, 0.0) {
		fs = append(fs, patternStopFunction(value(&stops[0]), value(&stops[0])))
		encode = append(encode, 0, 1)
		bounds = append(bounds, stops[0].Pos)
	}
	for i := range n - 1 {
		fs = append(fs, patternStopFunction(value(&stops[i]), value(&stops[i+1])))
		encode = append(encode, 0, 1)
		if i != 0 {
			bounds = append(bounds, stops[i].Pos) // the start of this function
		}
	}
	if !ppath.Equal(stops[n-1].Pos, 1.0) {
		fs = append(fs, patternStopFunction(value(&stops[n-1]), value(&stops[n-1])))
		encode = append(encode, 0, 1)
		bounds = append(bounds, stops[n-1].Pos)
	}
	if len(fs) == 1 {
		return fs[0].(pdfDict)
	}
	return pdfDict{
		"FunctionType": 3,
		"Domain":       pdfArray{0, 1},
		"Encode":       encode,
		"Bounds":       bounds,
		"Functions":    fs,
	}
}

// patternStopFunction returns the function interpolating between the
// given values of two stops.
func patternStopFunction(v0, v1 pdfArray) pdfDict {
	return pdfDict{
		"FunctionType": 2,
		"Domain":       pdfArray{0, 1},
		"N":            1,
		"C0":           v0,
		"C1":           v1,
	}
}
