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
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
)

// Path renders a path to the canvas using a style and an
// individual matrix (needed for fill)
func (r *PDF) Path(path ppath.Path, style *styles.Paint, bounds math32.Box2, tr math32.Matrix2) {
	// PDFs don't support the arcs joiner, miter joiner (not clipped),
	// or miter joiner (clipped) with non-bevel fallback
	strokeUnsupported := false
	if style.Stroke.Join == ppath.JoinArcs {
		strokeUnsupported = true
	} else if style.Stroke.Join == ppath.JoinMiter {
		if style.Stroke.MiterLimit == 0 {
			strokeUnsupported = true
		}
		// } else if _, ok := miter.GapJoiner.(canvas.BevelJoiner); !ok {
		// 	strokeUnsupported = true
		// }
	}
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
	data := path.Clone().Transform(tr).ToPDF()
	if 1 < len(data) && data[len(data)-1] == 'h' {
		data = data[:len(data)-2]
		closed = true
	}

	if style.HasStroke() && strokeUnsupported {
		// todo: handle with optional inclusion of stroke function as _ import
		/*	// style.HasStroke() && strokeUnsupported
			if style.HasFill() {
				r.w.SetFill(style.Fill)
				r.w.Write([]byte(" "))
				r.w.Write([]byte(data))
				r.w.Write([]byte(" f"))
				if style.Fill.Rule == canvas.EvenOdd {
					r.w.Write([]byte("*"))
				}
			}

			// stroke settings unsupported by PDF, draw stroke explicitly
			if style.IsDashed() {
				path = path.Dash(style.DashOffset, style.Dashes...)
			}
			path = path.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner, canvas.Tolerance)

			r.w.SetFill(style.Stroke)
			r.w.Write([]byte(" "))
			r.w.Write([]byte(path.Transform(m).ToPDF()))
			r.w.Write([]byte(" f"))
		*/
		// return
	}
	if style.HasFill() && !style.HasStroke() {
		r.w.SetFill(&style.Fill, bounds, tr)
		r.w.Write([]byte(" "))
		r.w.Write([]byte(data))
		r.w.Write([]byte(" f"))
		if style.Fill.Rule == ppath.EvenOdd {
			r.w.Write([]byte("*"))
		}
	} else if !style.HasFill() && style.HasStroke() {
		r.w.SetStroke(&stk)
		r.w.Write([]byte(" "))
		r.w.Write([]byte(data))
		if closed {
			r.w.Write([]byte(" s"))
		} else {
			r.w.Write([]byte(" S"))
		}
		if style.Fill.Rule == ppath.EvenOdd {
			r.w.Write([]byte("*"))
		}
	} else if style.HasFill() && style.HasStroke() {
		// sameAlpha := style.Fill.IsColor() && style.Stroke.IsColor() && style.Fill.Color.A == style.Stroke.Color.A
		// todo:
		sameAlpha := true
		if sameAlpha {
			r.w.SetFill(&style.Fill, bounds, tr)
			r.w.SetStroke(&style.Stroke)
			r.w.Write([]byte(" "))
			r.w.Write([]byte(data))
			if closed {
				r.w.Write([]byte(" b"))
			} else {
				r.w.Write([]byte(" B"))
			}
			if style.Fill.Rule == ppath.EvenOdd {
				r.w.Write([]byte("*"))
			}
		} else {
			r.w.SetFill(&style.Fill, bounds, tr)
			r.w.Write([]byte(" "))
			r.w.Write([]byte(data))
			r.w.Write([]byte(" f"))
			if style.Fill.Rule == ppath.EvenOdd {
				r.w.Write([]byte("*"))
			}

			r.w.SetStroke(&style.Stroke)
			r.w.Write([]byte(" "))
			r.w.Write([]byte(data))
			if closed {
				r.w.Write([]byte(" s"))
			} else {
				r.w.Write([]byte(" S"))
			}
			if style.Fill.Rule == ppath.EvenOdd {
				r.w.Write([]byte("*"))
			}
		}
	}
}

// SetFill sets the fill style values where different from current.
// The bounds and matrix are required for gradient fills: pass identity
// if not available.
func (w *pdfPage) SetFill(fill *styles.Fill, bounds math32.Box2, m math32.Matrix2) {
	csty := w.style()
	if csty.Fill.Color != fill.Color || csty.Fill.Opacity != fill.Opacity {
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
	case *gradient.Linear:
		fmt.Fprintf(w, " /Pattern cs /%v scn", w.gradientPattern(x, bounds, m))
	case *gradient.Radial:
		fmt.Fprintf(w, " /Pattern cs /%v scn", w.gradientPattern(x, bounds, m))
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
		// fmt.Fprintf(w, " /Pattern cs /%v scn", w.gradientPattern(stroke.Gradient))
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

func (w *pdfPage) gradientPattern(gr gradient.Gradient, bounds math32.Box2, m math32.Matrix2) pdfName {
	// fbox := sc.GetPathExtent()
	// lastRenderBBox := image.Rectangle{Min: image.Point{fbox.Min.X.Floor(), fbox.Min.Y.Floor()},
	// 	Max: image.Point{fbox.Max.X.Ceil(), fbox.Max.Y.Ceil()}}
	cum := w.Cumulative().Mul(m)
	gr.Update(1, bounds, cum)
	// TODO: support patterns/gradients with alpha channel
	shading := pdfDict{
		"ColorSpace": pdfName("DeviceRGB"),
	}
	switch g := gr.(type) {
	case *gradient.Linear:
		s, e := g.TransformedAxis()
		shading["ShadingType"] = 2
		shading["Coords"] = pdfArray{s.X, s.Y, e.X, e.Y}
		shading["Function"] = patternStopsFunction(g.Stops)
		shading["Extend"] = pdfArray{true, true}
	case *gradient.Radial:
		c, f, rad := g.TransformedCoords()
		// r := 0.5 * (math32.Abs(rad.X) + math32.Abs(rad.Y))
		r := max(math32.Abs(rad.X), math32.Abs(rad.Y))
		// fmt.Println("c:", c, "ctr:", g.Center)
		shading["ShadingType"] = 3
		shading["Coords"] = pdfArray{f.X, f.Y, 0, c.X, c.Y, r}
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
	n := len(stops)
	if len(stops) < 2 {
		return pdfDict{}
	}

	fs := pdfArray{}
	encode := pdfArray{}
	bounds := pdfArray{}
	if !ppath.Equal(stops[0].Pos, 0.0) {
		fs = append(fs, patternStopFunction(stops[0], stops[0]))
		encode = append(encode, 0, 1)
		bounds = append(bounds, stops[0].Pos)
	}
	for i := range n - 1 {
		fs = append(fs, patternStopFunction(stops[i], stops[i+1]))
		encode = append(encode, 0, 1)
		if i != 0 {
			bounds = append(bounds, stops[1].Pos)
		}
	}
	if !ppath.Equal(stops[n-1].Pos, 1.0) {
		fs = append(fs, patternStopFunction(stops[n-1], stops[n-1]))
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
