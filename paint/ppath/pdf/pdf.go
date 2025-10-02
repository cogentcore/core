// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package pdf

import (
	"image"
	"io"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/shaped"
)

// type Options struct {
// 	Compress    bool
// 	SubsetFonts bool
// 	canvas.ImageEncoding
// }
//
// var DefaultOptions = Options{
// 	Compress:      true,
// 	SubsetFonts:   true,
// 	ImageEncoding: canvas.Lossless,
// }

// PDF is a portable document format renderer.
type PDF struct {
	w             *pdfPageWriter
	width, height float32
	// opts          *Options
}

// New returns a portable document format (PDF) renderer.
// The size is in points.
func New(w io.Writer, width, height float32, un *units.Context) *PDF {
	// if opts == nil {
	// 	defaultOptions := DefaultOptions
	// 	opts = &defaultOptions
	// }

	page := newPDFWriter(w, un).NewPage(width, height)
	// page.pdf.SetCompression(opts.Compress)
	// page.pdf.SetFontSubsetting(opts.SubsetFonts)
	return &PDF{
		w:      page,
		width:  width,
		height: height,
		// opts:   opts,
	}
}

// SetImageEncoding sets the image encoding to Loss or Lossless.
// func (r *PDF) SetImageEncoding(enc canvas.ImageEncoding) {
// 	r.opts.ImageEncoding = enc
// }

// SetInfo sets the document's title, subject, keywords, author and creator.
func (r *PDF) SetInfo(title, subject, keywords, author, creator string) {
	r.w.pdf.SetTitle(title)
	r.w.pdf.SetSubject(subject)
	r.w.pdf.SetKeywords(keywords)
	r.w.pdf.SetAuthor(author)
	r.w.pdf.SetCreator(creator)
}

// SetLang sets the document's language. It must adhere the RFC 3066 specification on Language-Tag, eg. es-CL.
func (r *PDF) SetLang(lang string) {
	r.w.pdf.SetLang(lang)
}

// NewPage starts adds a new page where further rendering will be written to.
func (r *PDF) NewPage(width, height float32) {
	r.w = r.w.pdf.NewPage(width, height)
}

// AddLink adds a link to the PDF document.
func (r *PDF) AddLink(uri string, rect math32.Box2) {
	r.w.AddURIAction(uri, rect)
}

// Close finished and closes the PDF.
func (r *PDF) Close() error {
	return r.w.pdf.Close()
}

// Size returns the size of the canvas in millimeters.
func (r *PDF) Size() (float32, float32) {
	return r.width, r.height
}

// AddLayer defines a layer that can be shown or hidden when the document is
// displayed. name specifies the layer name that the document reader will
// display in the layer list. visible specifies whether the layer will be
// initially visible. The return value is an integer ID that is used in a call
// to BeginLayer().
func (r *PDF) AddLayer(name string, visible bool) (layerID int) {
	return r.w.pdf.AddLayer(name, visible)
}

// BeginLayer is called to begin adding content to the specified layer. All
// content added to the page between a call to BeginLayer and a call to
// EndLayer is added to the layer specified by id. See AddLayer for more
// details.
func (r *PDF) BeginLayer(id int) {
	r.w.pdf.BeginLayer(id)
}

// EndLayer is called to stop adding content to the currently active layer. See
// BeginLayer for more details.
func (r *PDF) EndLayer() {
	r.w.pdf.EndLayer()
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *PDF) RenderPath(path ppath.Path, style *styles.Paint, m math32.Matrix2) {
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
	scale := math32.Sqrt(math32.Abs(m.Det()))
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
	data := path.Clone().Transform(m).ToPDF()
	if 1 < len(data) && data[len(data)-1] == 'h' {
		data = data[:len(data)-2]
		closed = true
	}

	if style.HasStroke() && strokeUnsupported { // todo
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
		return
	}
	if style.HasFill() && !style.HasStroke() {
		r.w.SetFill(&style.Fill)
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
			r.w.SetFill(&style.Fill)
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
		}
		/*
			 else {
				r.w.SetFill(style.Fill)
				r.w.Write([]byte(" "))
				r.w.Write([]byte(data))
				r.w.Write([]byte(" f"))
				if style.Fill.Rule == ppath.EvenOdd {
					r.w.Write([]byte("*"))
				}

				r.w.SetStroke(style.Stroke)
				r.w.SetLineWidth(style.StrokeWidth)
				r.w.SetLineCap(style.StrokeCapper)
				r.w.SetLineJoin(style.StrokeJoiner)
				r.w.SetDashes(style.DashOffset, style.Dashes)
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
		*/
	}
}

// RenderText renders a text object to the canvas using a transformation matrix,
// (the translation component specifies the starting offset)
func (r *PDF) RenderText(text *shaped.Lines, m math32.Matrix2) {
	// text.WalkDecorations(func(fill canvas.Paint, p *canvas.Path) {
	// 	style := canvas.DefaultStyle
	// 	style.Fill = fill
	// 	r.RenderPath(p, style, m)
	// })

	// todo: copy from other render cases
	// text.WalkSpans(func(x, y float32, span canvas.TextSpan) {
	// 	if span.IsText() {
	// 		style := canvas.DefaultStyle
	// 		style.Fill = span.Face.Fill
	//
	// 		r.w.StartTextObject()
	// 		r.w.SetFill(span.Face.Fill)
	// 		r.w.SetFont(span.Face.Font, span.Face.Size, span.Direction)
	// 		r.w.SetTextPosition(m.Translate(x, y).Shear(span.Face.FauxItalic, 0.0))
	//
	// 		if 0.0 < span.Face.FauxBold {
	// 			r.w.SetTextRenderMode(2)
	// 			r.w.SetStroke(span.Face.Fill)
	// 			fmt.Fprintf(r.w, " %v w", dec(span.Face.FauxBold*2.0))
	// 		} else {
	// 			r.w.SetTextRenderMode(0)
	// 		}
	// 		r.w.WriteText(text.WritingMode, span.Glyphs)
	// 		r.w.EndTextObject()
	// 	} else {
	// 		for _, obj := range span.Objects {
	// 			obj.Canvas.RenderViewTo(r, m.Mul(obj.View(x, y, span.Face)))
	// 		}
	// 	}
	// })
}

// RenderImage renders an image to the canvas using a transformation matrix.
func (r *PDF) RenderImage(img image.Image, m math32.Matrix2) {
	r.w.DrawImage(img, m)
}
