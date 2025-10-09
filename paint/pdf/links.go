// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pdf

import (
	"cogentcore.org/core/math32"
)

// AddAnchor adds a uniquely-named link anchor location.
// The position is in "default user space" coordinates = standard page coordinates,
// without the current CTM transform.// This function will handle the base page
// transform for scaling and flipping of coordinates to top-left system.
func (w *pdfPage) AddAnchor(name string, pos math32.Vector2) {
	ms := math32.Scale2D(w.pdf.globalScale, w.pdf.globalScale)
	pos = ms.MulVector2AsPoint(pos)
	if w.pdf.anchors == nil {
		w.pdf.anchors = make(pdfMap)
	}
	// pageNo replaced at end with ref:
	w.pdf.anchors[name] = pdfArray{w.pageNo, pdfName("XYZ"), 0, w.height - pos.Y, 0}
	// fmt.Println("anchor:", w.pageNo, name, pos)
}

// AddLink adds a link annotation. The rect is in "default user space"
// coordinates = standard page coordinates, without the current CTM transform.
// This function will handle the base page transform for scaling and
// flipping of coordinates to top-left system.
func (w *pdfPage) AddLink(uri string, rect math32.Box2) {
	ms := math32.Scale2D(w.pdf.globalScale, w.pdf.globalScale)
	rect = rect.MulMatrix2(ms)
	isLocal := false
	if uri[0] == '#' { // local anchor actions
		uri = uri[1:]
		isLocal = true
	}
	annot := pdfDict{
		"Type":    pdfName("Annot"),
		"Subtype": pdfName("Link"),
		"Border":  pdfArray{0, 0, 0},
		"Rect":    pdfArray{rect.Min.X, w.height - rect.Max.Y, rect.Max.X, w.height - rect.Min.Y},
	}
	if isLocal {
		annot["Dest"] = uri
	} else {
		annot["A"] = pdfDict{
			"S":            pdfName("URI"),
			pdfName("URI"): uri,
		}
	}
	w.annots = append(w.annots, annot)
}
