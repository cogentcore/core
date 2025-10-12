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

////////  outline

// pdfOutline is one outline entry
type pdfOutline struct {
	name  string
	level int
	page  int
	ypos  float32 // inverted y position

	parent int // index into outline list
	count  int
	prev   int
	next   int
	first  int
	last   int
}

// AddOutline adds an outline element. level must be > 0
func (w *pdfPage) AddOutline(name string, level int, ypos float32) {
	ol := &pdfOutline{name: name, level: level, page: w.pageNo, ypos: w.height - w.pdf.globalScale*ypos, prev: -1, next: -1, first: -1, last: -1}
	if len(w.pdf.outlines) == 0 {
		or := &pdfOutline{name: "Contents", level: 0, page: 0, ypos: 0, prev: -1, next: -1, first: -1, last: -1}
		w.pdf.outlines = append(w.pdf.outlines, or)
	}
	w.pdf.outlines = append(w.pdf.outlines, ol)
}

// writeOutlines outputs all the outline elements, and returns the ref to the first one.
func (w *pdfWriter) writeOutlines() pdfRef {
	lastLev := make(map[int]int)
	level := 0
	n := len(w.outlines)
	// note: this logic from fpdf
	for i, o := range w.outlines {
		if o.level > 0 {
			parent := lastLev[o.level-1]
			o.parent = parent
			w.outlines[parent].last = i
			w.outlines[parent].count++
			if o.level > level {
				w.outlines[parent].first = i
			}
		} else {
			o.parent = n
		}
		if o.level <= level && i > 0 {
			prev := lastLev[o.level]
			w.outlines[prev].next = i
			o.prev = prev
		}
		lastLev[o.level] = i
		level = o.level
	}
	var refs pdfArray
	firstRef := pdfRef(len(w.objOffsets) + 1)
	for i, o := range w.outlines {
		od := pdfDict{
			"Title":  o.name,
			"Parent": firstRef + pdfRef(o.parent),
			"Dest":   pdfArray{w.pages[o.page], pdfName("XYZ"), 0, o.ypos, 0},
			"Count":  o.count,
		}
		if o.prev != -1 {
			od[pdfName("Prev")] = firstRef + pdfRef(o.prev)
		}
		if o.next != -1 {
			od[pdfName("Next")] = firstRef + pdfRef(o.next)
		}
		if o.first != -1 {
			od[pdfName("First")] = firstRef + pdfRef(o.first)
		}
		if o.last != -1 {
			od[pdfName("Last")] = firstRef + pdfRef(o.last)
		}
		if i == 0 {
			delete(od, "Parent")
			delete(od, "Dest")
		}
		ref := w.writeObject(od)
		refs = append(refs, ref)
	}
	return firstRef
}
