// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"math"

	"github.com/chewxy/math32"
	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// ColorMap maps a value onto a color by interpolating between a list of colors
// defining a spectrum
type ColorMap struct {
	Name   string
	Colors []gi.Color
}

// Map returns color for normalized value in range 0-1
func (cm *ColorMap) Map(val float64) gi.Color {
	nc := len(cm.Colors)
	if nc < 2 {
		return gi.Color{}
	}
	if val <= 0 {
		return cm.Colors[0]
	} else if val >= 1 {
		return cm.Colors[nc-1]
	}
	ival := val * float64(nc-1)
	lidx := math.Floor(ival)
	uidx := math.Ceil(ival)
	if lidx == uidx {
		return cm.Colors[int(lidx)]
	}
	cmix := ival - lidx
	lclr := cm.Colors[int(lidx)]
	uclr := cm.Colors[int(uidx)]
	return lclr.Blend(float32(cmix)*100, uclr)
}

// StdColorMaps is a list of standard color maps
var StdColorMaps = map[string]*ColorMap{
	"ColdHot": &ColorMap{"ColdHot", []gi.Color{{0, 255, 255, 255}, {0, 0, 255, 255}, {127, 127, 127, 255}, {255, 0, 0, 255}, {255, 255, 0, 255}}},
	"Jet":     &ColorMap{"Jet", []gi.Color{{0, 0, 127, 255}, {0, 0, 255, 255}, {0, 127, 255, 255}, {0, 255, 255, 255}, {127, 255, 127, 255}, {255, 255, 0, 255}, {255, 127, 0, 255}, {255, 0, 0, 255}, {127, 0, 0, 255}}},
}

// AvailColorMaps is the list of all available color maps
var AvailColorMaps = map[string]*ColorMap{}

func init() {
	for k, v := range StdColorMaps {
		AvailColorMaps[k] = v
	}
}

// ColorMapName provides a gui chooser of maps in AvailColorMaps
type ColorMapName string

/////////////////////////////////////////////////////////////////////////////
//  ColorMapView

// ColorMapView is a widget that displays a ColorMap.
// Note that this is not a ValueView widget
type ColorMapView struct {
	gi.WidgetBase
	Orient gi.Dims2D `desc:"orientation along which to display the spectrum"`
	Map    *ColorMap `desc:"the colormap that we view"`
}

var KiT_ColorMapView = kit.Types.AddType(&ColorMapView{}, nil)

// AddNewColorMapView adds a new colorview to given parent node, with given name.
func AddNewColorMapView(parent ki.Ki, name string, cmap *ColorMap) *ColorMapView {
	cv := parent.AddNewChild(KiT_ColorMapView, name).(*ColorMapView)
	cv.Map = cmap
	return cv
}

// SetColorMap sets the color map and triggers a display update
func (cv *ColorMapView) SetColorMap(cmap *ColorMap) {
	cv.Map = cmap
	cv.UpdateSig()
}

func (cv *ColorMapView) RenderColorMap() {
	if cv.Map == nil {
		cv.Map = StdColorMaps["ColdHot"]
	}
	rs := &cv.Viewport.Render
	rs.Lock()
	pc := &rs.Paint

	pos := cv.LayData.AllocPos
	sz := cv.LayData.AllocSize

	lsz := sz.Dim(cv.Orient)
	inc := math32.Ceil(lsz / 100)
	if inc < 2 {
		inc = 2
	}
	for p := float32(0); p < lsz; p += inc {
		val := p / (lsz - 1)
		clr := cv.Map.Map(float64(val))
		if cv.Orient == gi.X {
			pr := pos
			pr.X += p
			sr := sz
			sr.X = inc
			pc.FillBoxColor(rs, pr, sr, clr)
		} else {
			pr := pos
			pr.Y += p
			sr := sz
			sr.Y = inc
			pc.FillBoxColor(rs, pr, sr, clr)
		}
	}
	rs.Unlock()
}

func (cv *ColorMapView) Render2D() {
	if cv.FullReRenderIfNeeded() {
		return
	}
	if cv.PushBounds() {
		cv.This().(gi.Node2D).ConnectEvents2D()
		cv.RenderColorMap()
		cv.Render2DChildren()
		cv.PopBounds()
	} else {
		cv.DisconnectAllEvents(gi.RegPri)
	}
}
