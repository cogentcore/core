// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package pdf

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"log/slog"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/stack"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"golang.org/x/text/encoding/charmap"
)

type pdfPage struct {
	*bytes.Buffer
	pdf           *pdfWriter
	width, height float32
	resources     pdfDict
	annots        pdfArray

	graphicsStates map[float32]pdfName
	stack          stack.Stack[*context]
	inTextObject   bool
	textPosition   math32.Vector2
	textCharSpace  float32
	textRenderMode int
}

func (w *pdfPage) writePage(parent pdfRef) pdfRef {
	b := w.Bytes()
	if 0 < len(b) && b[0] == ' ' {
		b = b[1:]
	}
	stream := pdfStream{
		dict:   pdfDict{},
		stream: b,
	}
	if w.pdf.compress {
		stream.dict["Filter"] = pdfFilterFlate
	}
	contents := w.pdf.writeObject(stream)
	page := pdfDict{
		"Type":      pdfName("Page"),
		"Parent":    parent,
		"MediaBox":  pdfArray{0.0, 0.0, w.width, w.height},
		"Resources": w.resources,
		"Group": pdfDict{
			"Type": pdfName("Group"),
			"S":    pdfName("Transparency"),
			"I":    true,
			"CS":   pdfName("DeviceRGB"),
		},
		"Contents": contents,
	}
	if 0 < len(w.annots) {
		page["Annots"] = w.annots
	}
	return w.pdf.writeObject(page)
}

// AddAnnotation adds an annotation. The rect is in "default user space"
// coordinates = standard page coordinates, without the current CTM transform.
// This function will handle the base page transform for scaling and
// flipping of coordinates to top-left system.
func (w *pdfPage) AddURIAction(uri string, rect math32.Box2) {
	ms := math32.Scale2D(w.pdf.globalScale, w.pdf.globalScale)
	rect = rect.MulMatrix2(ms)
	annot := pdfDict{
		"Type":     pdfName("Annot"),
		"Subtype":  pdfName("Link"),
		"Border":   pdfArray{0, 0, 0},
		"Rect":     pdfArray{rect.Min.X, w.height - rect.Max.Y, rect.Max.X, w.height - rect.Min.Y},
		"Contents": uri,
		"A": pdfDict{
			"S":   pdfName("URI"),
			"URI": uri,
		},
	}
	w.annots = append(w.annots, annot)
}

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

// SetFillColor sets the filling color (image).
func (w *pdfPage) SetFillColor(fill *styles.Fill) {
	switch x := fill.Color.(type) {
	// todo: pattern, image
	case *gradient.Linear:
	case *gradient.Radial:
		// TODO: should we unset cs?
		// fmt.Fprintf(w, " /Pattern cs /%v scn", w.getPattern(fill.Gradient))
	case *image.Uniform:
		var clr color.RGBA
		if x != nil {
			clr = colors.ApplyOpacity(colors.AsRGBA(x), fill.Opacity)
		}
		a := float32(clr.A) / 255.0
		if clr.R == clr.G && clr.R == clr.B {
			fmt.Fprintf(w, " %v g", dec(float32(clr.R)/255.0/a))
		} else {
			fmt.Fprintf(w, " %v %v %v rg", dec(float32(clr.R)/255.0/a), dec(float32(clr.G)/255.0/a), dec(float32(clr.B)/255.0/a))
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
			fmt.Fprintf(w, " %v G", dec(float32(clr.R)/255.0/a))
		} else {
			fmt.Fprintf(w, " %v %v %v RG", dec(float32(clr.R)/255.0/a), dec(float32(clr.G)/255.0/a), dec(float32(clr.B)/255.0/a))
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

// SetFont sets the font.
func (w *pdfPage) SetFont(sty *rich.Style, tsty *text.Style) error {
	if !w.inTextObject {
		return errors.Log(errors.New("pdfWriter: must be in text object"))
	}
	size := tsty.FontHeight(sty) // * w.pdf.globalScale
	ref := w.pdf.getFont(sty, tsty)
	if _, ok := w.resources["Font"]; !ok {
		w.resources["Font"] = pdfDict{}
	} else {
		for name, fontRef := range w.resources["Font"].(pdfDict) {
			if ref == fontRef {
				fmt.Fprintf(w, " /%v %v Tf", name, dec(size))
				return nil
			}
		}
	}

	name := pdfName(fmt.Sprintf("F%d", len(w.resources["Font"].(pdfDict))))
	w.resources["Font"].(pdfDict)[name] = ref
	fmt.Fprintf(w, " /%v %v Tf", name, dec(size))
	return nil
}

// SetTextPosition sets the text offset position.
func (w *pdfPage) SetTextPosition(off math32.Vector2) error {
	if !w.inTextObject {
		return errors.Log(errors.New("pdfWriter: must be in text object"))
	}
	do := off.Sub(w.textPosition)
	// and finally apply an offset from there, in reverse for Y
	fmt.Fprintf(w, " %v %v Td", dec(do.X), dec(-do.Y))
	w.textPosition = off
	return nil
}

// SetTextRenderMode sets the text rendering mode.
// 0 = fill text, 1 = stroke text, 2 = fill, then stroke.
// higher numbers support clip path.
func (w *pdfPage) SetTextRenderMode(mode int) error {
	if !w.inTextObject {
		return errors.Log(errors.New("pdfWriter: must be in text object"))
	}
	fmt.Fprintf(w, " %d Tr", mode)
	w.textRenderMode = mode
	return nil
}

// SetTextCharSpace sets the text character spacing.
func (w *pdfPage) SetTextCharSpace(space float32) error {
	if !w.inTextObject {
		return errors.Log(errors.New("pdfWriter: must be in text object"))
	}
	fmt.Fprintf(w, " %v Tc", dec(space))
	w.textCharSpace = space
	return nil
}

// StartTextObject starts a text object, adding to the graphics
// CTM transform matrix as given by the arg, and setting an inverting
// text transform, so text is rendered upright.
func (w *pdfPage) StartTextObject(m math32.Matrix2) error {
	if w.inTextObject {
		return errors.Log(errors.New("pdfWriter: already in text object"))
	}
	// set the graphics transform to m first
	w.PushTransform(m)
	fmt.Fprintf(w, " BT")
	// then apply an inversion text matrix
	tm := math32.Scale2D(1, -1)
	fmt.Fprintf(w, " %s Tm", mat2(tm))
	w.inTextObject = true
	w.textPosition = math32.Vector2{}
	return nil
}

// EndTextObject ends a text object.
func (w *pdfPage) EndTextObject() error {
	if !w.inTextObject {
		return errors.Log(errors.New("pdfWriter: must be in text object"))
	}
	fmt.Fprintf(w, " ET")
	w.PopStack()
	w.inTextObject = false
	return nil
}

// WriteText writes text using current text style.
func (w *pdfPage) WriteText(tx string) error {
	if !w.inTextObject {
		return errors.Log(errors.New("pdfWriter: must be in text object"))
	}
	if len(tx) == 0 {
		return nil
	}

	first := true
	write := func(s string) {
		if first {
			fmt.Fprintf(w, "(")
			first = false
		} else {
			fmt.Fprintf(w, " (")
		}
		rs := []rune(s)
		for _, r := range rs {
			c, ok := charmap.Windows1252.EncodeRune(r)
			if !ok {
				if '\u2000' <= r && r <= '\u200A' {
					c = ' '
				}
			}
			switch c {
			case '\n':
				w.WriteByte('\\')
				w.WriteByte('n')
			case '\r':
				w.WriteByte('\\')
				w.WriteByte('r')
			case '\t':
				w.WriteByte('\\')
				w.WriteByte('t')
			case '\b':
				w.WriteByte('\\')
				w.WriteByte('b')
			case '\f':
				w.WriteByte('\\')
				w.WriteByte('f')
			case '\\', '(', ')':
				w.WriteByte('\\')
				w.WriteByte(c)
			default:
				w.WriteByte(c)
			}
		}
		fmt.Fprintf(w, ")")
	}

	// position := w.textPosition
	// if glyphs, ok := TJ[0].([]canvasText.Glyph); ok && 0 < len(glyphs) && mode != ppath.HorizontalTB && !glyphs[0].Vertical {
	// 	glyphRotation, glyphOffset := glyphs[0].Rotation(), glyphs[0].YOffset-int32(glyphs[0].SFNT.Head.UnitsPerEm/2)
	// 	if glyphRotation != canvasText.NoRotation || glyphOffset != 0 {
	// 		w.SetTextPosition(position.Rotate(float32(glyphRotation)).Translate(0.0, glyphs[0].Size/float32(glyphs[0].SFNT.Head.UnitsPerEm)*mmPerPt*float32(glyphOffset)))
	// 	}
	// }

	// f := 1000.0 / float32(w.font.SFNT.Head.UnitsPerEm)
	fmt.Fprintf(w, "[")
	write(tx)

	// for _, tj := range TJ {
	// 	switch val := tj.(type) {
	// 	case []canvasText.Glyph:
	// 		i := 0
	// 		for j, glyph := range val {
	// 			if mode == ppath.HorizontalTB || !glyph.Vertical {
	// 				origXAdvance := int32(w.font.SFNT.GlyphAdvance(glyph.ID))
	// 				if glyph.XAdvance != origXAdvance {
	// 					write(val[i : j+1])
	// 					fmt.Fprintf(w, " %d", -int(f*float32(glyph.XAdvance-origXAdvance)+0.5))
	// 					i = j + 1
	// 				}
	// 			} else {
	// 				origYAdvance := -int32(w.font.SFNT.GlyphVerticalAdvance(glyph.ID))
	// 				if glyph.YAdvance != origYAdvance {
	// 					write(val[i : j+1])
	// 					fmt.Fprintf(w, " %d", -int(f*float32(glyph.YAdvance-origYAdvance)+0.5))
	// 					i = j + 1
	// 				}
	// 			}
	// 		}
	// 		write(val[i:])
	// 	case string:
	// 		i := 0
	// 		if mode == ppath.HorizontalTB {
	// 			var rPrev rune
	// 			for j, r := range val {
	// 				if i < j {
	// 					kern := w.font.SFNT.Kerning(w.font.SFNT.GlyphIndex(rPrev), w.font.SFNT.GlyphIndex(r))
	// 					if kern != 0 {
	// 						writeString(val[i:j])
	// 						fmt.Fprintf(w, " %d", -int(f*float32(kern)+0.5))
	// 						i = j
	// 					}
	// 				}
	// 				rPrev = r
	// 			}
	// 		}
	// 		writeString(val[i:])
	// 	case float32:
	// 		fmt.Fprintf(w, " %d", -int(val*1000.0/w.fontSize+0.5))
	// 	case int:
	// 		fmt.Fprintf(w, " %d", -int(float32(val)*1000.0/w.fontSize+0.5))
	// 	}
	// }
	fmt.Fprintf(w, "]TJ")
	return nil
}

// DrawImage embeds and draws an image, as a lossless (PNG)
func (w *pdfPage) DrawImage(img image.Image, m math32.Matrix2) {
	size := img.Bounds().Size()

	// add clipping path around image for smooth edges when rotating
	outerRect := math32.B2(0.0, 0.0, float32(size.X), float32(size.Y)).MulMatrix2(m)
	bl := m.MulVector2AsPoint(math32.Vector2{0, 0})
	br := m.MulVector2AsPoint(math32.Vector2{float32(size.X), 0})
	tl := m.MulVector2AsPoint(math32.Vector2{0, float32(size.Y)})
	tr := m.MulVector2AsPoint(math32.Vector2{float32(size.X), float32(size.Y)})
	fmt.Fprintf(w, " q %v %v %v %v re W n", dec(outerRect.Min.X), dec(outerRect.Min.Y), dec(outerRect.Size().X), dec(outerRect.Size().Y))
	fmt.Fprintf(w, " %v %v m %v %v l %v %v l %v %v l h W n", dec(bl.X), dec(bl.Y), dec(tl.X), dec(tl.Y), dec(tr.X), dec(tr.Y), dec(br.X), dec(br.Y))

	ref := w.embedImage(img)
	if _, ok := w.resources["XObject"]; !ok {
		w.resources["XObject"] = pdfDict{}
	}
	name := pdfName(fmt.Sprintf("Im%d", len(w.resources["XObject"].(pdfDict))))
	w.resources["XObject"].(pdfDict)[name] = ref

	m = m.Scale(float32(size.X), float32(size.Y))
	w.SetAlpha(1.0)
	fmt.Fprintf(w, " %s cm /%v Do Q", mat2(m), name)
}

// embedImage does a lossless image embedding.
func (w *pdfPage) embedImage(img image.Image) pdfRef {
	if ref, ok := w.pdf.images[img]; ok {
		return ref
	}

	var hasMask bool
	size := img.Bounds().Size()
	filter := pdfFilterFlate
	sp := img.Bounds().Min // starting point
	stream := make([]byte, size.X*size.Y*3)
	streamMask := make([]byte, size.X*size.Y)
	for y := size.Y - 1; y >= 0; y-- { // invert
		for x := range size.X {
			pi := (size.Y-1-y)*size.X + x
			i := pi * 3
			R, G, B, A := img.At(sp.X+x, sp.Y+y).RGBA()
			if A != 0 {
				stream[i+0] = byte((R * 65535 / A) >> 8)
				stream[i+1] = byte((G * 65535 / A) >> 8)
				stream[i+2] = byte((B * 65535 / A) >> 8)
				streamMask[pi] = byte(A >> 8)
			}
			if A>>8 != 255 {
				hasMask = true
			}
		}
	}

	dict := pdfDict{
		"Type":             pdfName("XObject"),
		"Subtype":          pdfName("Image"),
		"Width":            size.X,
		"Height":           size.Y,
		"ColorSpace":       pdfName("DeviceRGB"),
		"BitsPerComponent": 8,
		"Interpolate":      true,
		"Filter":           filter,
	}

	if hasMask {
		dict["SMask"] = w.pdf.writeObject(pdfStream{
			dict: pdfDict{
				"Type":             pdfName("XObject"),
				"Subtype":          pdfName("Image"),
				"Width":            size.X,
				"Height":           size.Y,
				"ColorSpace":       pdfName("DeviceGray"),
				"BitsPerComponent": 8,
				"Interpolate":      true,
				"Filter":           pdfFilterFlate,
			},
			stream: streamMask,
		})
	}

	ref := w.pdf.writeObject(pdfStream{
		dict:   dict,
		stream: stream,
	})
	w.pdf.images[img] = ref
	return ref
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

/*
func (w *pdfPage) getPattern(gradient ppath.Gradient) pdfName {
	// TODO: support patterns/gradients with alpha channel
	shading := pdfDict{
		"ColorSpace": pdfName("DeviceRGB"),
	}
	if g, ok := gradient.(*ppath.LinearGradient); ok {
		shading["ShadingType"] = 2
		shading["Coords"] = pdfArray{g.Start.X, g.Start.Y, g.End.X, g.End.Y}
		shading["Function"] = patternStopsFunction(g.Stops)
		shading["Extend"] = pdfArray{true, true}
	} else if g, ok := gradient.(*ppath.RadialGradient); ok {
		shading["ShadingType"] = 3
		shading["Coords"] = pdfArray{g.C0.X, g.C0.Y, g.R0, g.C1.X, g.C1.Y, g.R1}
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

func patternStopsFunction(stops ppath.Stops) pdfDict {
	if len(stops) < 2 {
		return pdfDict{}
	}

	fs := []pdfDict{}
	encode := pdfArray{}
	bounds := pdfArray{}
	if !ppath.Equal(stops[0].Offset, 0.0) {
		fs = append(fs, patternStopFunction(stops[0], stops[0]))
		encode = append(encode, 0, 1)
		bounds = append(bounds, stops[0].Offset)
	}
	for i := 0; i < len(stops)-1; i++ {
		fs = append(fs, patternStopFunction(stops[i], stops[i+1]))
		encode = append(encode, 0, 1)
		if i != 0 {
			bounds = append(bounds, stops[1].Offset)
		}
	}
	if !ppath.Equal(stops[len(stops)-1].Offset, 1.0) {
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

func patternStopFunction(s0, s1 ppath.Stop) pdfDict {
	a0 := float32(s0.Color.A) / 255.0
	a1 := float32(s1.Color.A) / 255.0
	return pdfDict{
		"FunctionType": 2,
		"Domain":       pdfArray{0, 1},
		"N":            1,
		"C0":           pdfArray{float32(s0.Color.R) / 255.0 / a0, float32(s0.Color.G) / 255.0 / a0, float32(s0.Color.B) / 255.0 / a0},
		"C1":           pdfArray{float32(s1.Color.R) / 255.0 / a1, float32(s1.Color.G) / 255.0 / a1, float32(s1.Color.B) / 255.0 / a1},
	}
}

*/
