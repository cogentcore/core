// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/colors"
	"goki.dev/colors/colormap"
	"goki.dev/cursors"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/abilities"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/laser"
	"goki.dev/mat32/v2"
)

// ColorMapName provides a gui chooser of maps in AvailColorMaps
type ColorMapName string

/////////////////////////////////////////////////////////////////////////////
//  ColorMapView

// ColorMapView is a widget that displays a ColorMap.
// Note that this is not a Value widget
type ColorMapView struct {
	gi.Frame

	// Dim is the dimension on which to display the spectrum
	Dim mat32.Dims

	// the colormap that we view
	Map *colormap.Map
}

func (cv *ColorMapView) OnInit() {
	cv.HandleColorMapEvents()
	cv.ColorMapStyles()
}

func (cv *ColorMapView) ColorMapStyles() {
	cv.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable, abilities.Pressable)
		s.Cursor = cursors.Pointer
		s.Border.Radius = styles.BorderRadiusExtraSmall

		s.BackgroundColor.Gradient = colors.LinearGradient()
		for i := float32(0); i < 1; i += 0.01 {
			gc := cv.Map.Map(i)
			s.BackgroundColor.Gradient.AddStop(gc, i, 1)
		}
	})
}

// SetColorMap sets the color map and triggers a display update
func (cv *ColorMapView) SetColorMap(cmap *colormap.Map) *ColorMapView {
	cv.Map = cmap
	cv.SetNeedsRender(true)
	return cv
}

// SetColorMapAction sets the color map and triggers a display update
// and signals the ColorMapSig signal
func (cv *ColorMapView) SetColorMapAction(cmap *colormap.Map) *ColorMapView {
	cv.Map = cmap
	cv.SendChange()
	cv.SetNeedsRender(true)
	return cv
}

// ChooseColorMap pulls up a chooser to select a color map
func (cv *ColorMapView) ChooseColorMap() {
	sl := colormap.AvailMapsList()
	cur := ""
	if cv.Map != nil {
		cur = cv.Map.Name
	}
	si := 0
	d := gi.NewBody().AddTitle("Select a color map").AddText("Choose color map to use from among available list")
	NewSliceView(d).SetSlice(&sl).SetSelVal(cur).BindSelectDialog(&si)
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddOk(pw).OnClick(func(e events.Event) {
			if si >= 0 {
				nmap, ok := colormap.AvailMaps[sl[si]]
				if ok {
					cv.SetColorMapAction(nmap)
				}
			}
		})
	})
	d.NewFullDialog(cv).Run()
}

func (cv *ColorMapView) HandleColorMapEvents() {
	cv.HandleWidgetEvents()
	cv.OnClick(func(e events.Event) {
		cv.ChooseColorMap()
	})

}

////////////////////////////////////////////////////////////////////////////////////////
//  ColorMapValue

// Value registers ColorMapValue as the viewer of ColorMapName
func (mn ColorMapName) Value() Value {
	return &ColorMapValue{}
}

// ColorMapValue presents an button for displaying a ColorMapName and selecting
// meshes from a ChooserDialog
type ColorMapValue struct {
	ValueBase
}

func (vv *ColorMapValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *ColorMapValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none, click to select)"
	}
	bt.SetText(txt)
}

func (vv *ColorMapValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config()
	bt.OnClick(func(e events.Event) {
		if !vv.IsReadOnly() {
			vv.OpenDialog(vv.Widget, nil)
		}
	})
	vv.UpdateWidget()
}

func (vv *ColorMapValue) HasDialog() bool { return true }
func (vv *ColorMapValue) OpenDialog(ctx gi.Widget, fun func()) {
	OpenValueDialog(vv, ctx, fun, "Select a color map")
}

func (vv *ColorMapValue) ConfigDialog(d *gi.Body) (bool, func()) {
	sl := colormap.AvailMapsList()
	cur := laser.ToString(vv.Value.Interface())
	si := 0
	NewSliceView(d).SetSlice(&sl).SetSelVal(cur).BindSelectDialog(&si)
	return true, func() {
		if si >= 0 {
			vv.SetValue(sl[si])
			vv.UpdateWidget()
		}
	}
}
