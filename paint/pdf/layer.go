// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from codeberg.org/go-pdf/fpdf
// Copyright (c) 2023 The go-pdf Authors and Kurt Jung,
// under an MIT License.

package pdf

import "fmt"

// pdfLayer is one layer
type pdfLayer struct {
	name    string
	visible bool
	index   int
	ref     pdfRef
}

// pdfLayers is all the layers
type pdfLayers struct {
	list          []pdfLayer
	currentLayer  int
	openLayerPane bool
}

func (w *pdfWriter) layerInit() {
	w.layers.list = make([]pdfLayer, 0)
	w.layers.currentLayer = -1
	w.layers.openLayerPane = true
}

// AddLayer defines a layer that can be shown or hidden when the document is
// displayed. name specifies the layer name that the document reader will
// display in the layer list. visible specifies whether the layer will be
// initially visible. The return value is unique layer index that must be
// used in subsequent calls to BeginLayer.
func (w *pdfWriter) AddLayer(name string, visible bool) int {
	layno := len(w.layers.list)
	lay := pdfLayer{name: name, visible: visible, index: layno}
	lay.ref = w.writeObject(pdfDict{"Type": pdfName("OCG"), "Name": name})
	w.layers.list = append(w.layers.list, lay)
	return layno
}

// BeginLayer is called to begin adding content to the specified layer index
// in a given page. Layer must have already been added via AddLayer.
// All content added to the page between a call to BeginLayer and a call to
// EndLayer is added to the layer specified by id.
func (w *pdfPage) BeginLayer(id int) {
	w.EndLayer()
	if id < 0 || id >= len(w.pdf.layers.list) {
		return
	}
	ocn := fmt.Sprintf("oc%d", id)
	fmt.Fprintf(w, " /OC /%s BDC", ocn)
	w.pdf.layers.currentLayer = id
	if _, ok := w.resources["Properties"]; !ok {
		w.resources["Properties"] = pdfDict{}
	}
	l := &w.pdf.layers.list[id]
	w.resources["Properties"].(pdfDict)[pdfName(ocn)] = l.ref
}

// EndLayer is called to stop adding content to the currently active layer.
// See BeginLayer for more details.
func (w *pdfPage) EndLayer() {
	if w.pdf.layers.currentLayer < 0 {
		return
	}
	fmt.Fprintf(w, " EMC")
	w.pdf.layers.currentLayer = -1
}

// OpenLayerPane advises the document reader to open the layer pane when the
// document is initially displayed.
func (w *pdfWriter) OpenLayerPane() {
	w.layers.openLayerPane = true
}
