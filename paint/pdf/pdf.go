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
	"cogentcore.org/core/text/rich"
)

// UseStandardFonts sets the [rich.Settings] default fonts to the
// corresponding PDF defaults, so that text layout works correctly
// for the PDF rendering. The current settings are returned,
// and should be passed to [RestorePreviousFonts] when done.
func UseStandardFonts() rich.SettingsData {
	prev := rich.Settings
	rich.Settings.SansSerif = "Helvetica"
	rich.Settings.Serif = "Times"
	rich.Settings.Monospace = "Courier"
	return prev
}

// RestorePreviousFonts sets the [rich.Settings] default fonts
// to those returned from [UseStandardFonts]
func RestorePreviousFonts(s rich.SettingsData) {
	rich.Settings = s
}

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
	w             *pdfPage
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
	r.w.AddLink(uri, rect)
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

// BeginLayer is called to begin adding content to the specified layer.
// All content added to the page between a call to BeginLayer and a call to
// EndLayer is added to the layer specified by id. See AddLayer for more
// details.
func (r *PDF) BeginLayer(id int) {
	r.w.BeginLayer(id)
}

// EndLayer is called to stop adding content to the currently active layer.
// See BeginLayer for more details.
func (r *PDF) EndLayer() {
	r.w.EndLayer()
}

// PushStack adds a graphics stack push, which must
// be paired with a corresponding Pop.
func (r *PDF) PushStack() {
	r.w.PushStack()
}

// PopStack adds a graphics stack pop which must
// be paired with a corresponding Push.
func (r *PDF) PopStack() {
	r.w.PopStack()
}

// SetTransform adds a cm to set the current matrix transform (CMT).
func (r *PDF) SetTransform(m math32.Matrix2) {
	r.w.SetTransform(m)
}

// PushTransform adds a graphics stack push (q) and then
// cm to set the current matrix transform (CMT).
func (r *PDF) PushTransform(m math32.Matrix2) {
	r.PushStack()
	r.SetTransform(m)
}

// Cumulative returns the current cumulative transform.
func (r *PDF) Cumulative() math32.Matrix2 {
	return r.w.Cumulative()
}

// Path renders a path to the canvas using a style and an
// individual and cumulative transformation matrix (needed for fill)
func (r *PDF) Path(path ppath.Path, style *styles.Paint, bounds math32.Box2, tr, cum math32.Matrix2) {
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
		r.w.SetFill(&style.Fill, bounds, cum)
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
			r.w.SetFill(&style.Fill, bounds, cum)
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
			r.w.SetFill(&style.Fill, bounds, cum)
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

// Image renders an image to the canvas using a transformation matrix.
func (r *PDF) Image(img image.Image, m math32.Matrix2) {
	r.w.DrawImage(img, m)
}

// AddAnchor adds a uniquely-named link anchor location,
// which can then be a target for links.
func (r *PDF) AddAnchor(name string, pos math32.Vector2) {
	r.w.AddAnchor(name, pos)
}
