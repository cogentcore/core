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
	objNum  int // object number
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
	w.layers.openLayerPane = false
}

// AddLayer defines a layer that can be shown or hidden when the document is
// displayed. name specifies the layer name that the document reader will
// display in the layer list. visible specifies whether the layer will be
// initially visible. The return value is an integer ID that is used in a call
// to BeginLayer().
func (w *pdfWriter) AddLayer(name string, visible bool) (layerID int) {
	layerID = len(w.layers.list)
	w.layers.list = append(w.layers.list, pdfLayer{name: name, visible: visible})
	return
}

// BeginLayer is called to begin adding content to the specified layer. All
// content added to the page between a call to BeginLayer and a call to
// EndLayer is added to the layer specified by id. See AddLayer for more
// details.
func (w *pdfWriter) BeginLayer(id int) {
	w.EndLayer()
	if id >= 0 && id < len(w.layers.list) {
		w.write("/OC /OC%d BDC", id)
		w.layers.currentLayer = id
	}
}

// EndLayer is called to stop adding content to the currently active layer. See
// BeginLayer for more details.
func (w *pdfWriter) EndLayer() {
	if w.layers.currentLayer >= 0 {
		w.write("EMC")
		w.layers.currentLayer = -1
	}
}

// OpenLayerPane advises the document reader to open the layer pane when the
// document is initially displayed.
func (w *pdfWriter) OpenLayerPane() {
	w.layers.openLayerPane = true
}

func (w *pdfWriter) writeLayers() {
	for _, l := range w.layers.list {
		w.writeObject(&l)
	}
}

func (w *pdfWriter) writeLayerResourceDict() {
	if len(w.layers.list) == 0 {
		return
	}
	w.write("/Properties <<")
	for j, layer := range w.layers.list {
		w.write("/OC%d %d 0 R", j, layer.objNum)
	}
	w.write(">>")
}

func (w *pdfWriter) writeLayerCatalog() {
	if len(w.layers.list) == 0 {
		return
	}
	onStr := ""
	offStr := ""
	for _, layer := range w.layers.list {
		onStr += fmt.Sprintf("%d 0 R ", layer.objNum)
		if !layer.visible {
			offStr += fmt.Sprintf("%d 0 R ", layer.objNum)
		}
	}
	w.write("/OCProperties <</OCGs [%s] /D <</OFF [%s] /Order [%s]>>>>", onStr, offStr, onStr)
	if w.layers.openLayerPane {
		w.write("/PageMode /UseOC")
	}
}
