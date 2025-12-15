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

	"cogentcore.org/core/base/stack"
	"cogentcore.org/core/math32"
)

type pdfPage struct {
	*bytes.Buffer
	pdf           *pdfWriter
	pageNo        int
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

// DrawImage embeds and draws an image, as a lossless (PNG)
func (w *pdfPage) DrawImage(img image.Image, m math32.Matrix2) {
	size := img.Bounds().Size()

	// add clipping path around image for smooth edges when rotating
	outerRect := math32.B2(0.0, 0.0, float32(size.X), float32(size.Y)).MulMatrix2(m)
	bl := m.MulPoint(math32.Vector2{0, 0})
	br := m.MulPoint(math32.Vector2{float32(size.X), 0})
	tl := m.MulPoint(math32.Vector2{0, float32(size.Y)})
	tr := m.MulPoint(math32.Vector2{float32(size.X), float32(size.Y)})
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
