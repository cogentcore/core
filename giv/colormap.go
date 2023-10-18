// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/colors/colormap"
	"goki.dev/gi/v2/gi"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/ki/v2"
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
	gi.WidgetBase

	// orientation along which to display the spectrum
	Orient mat32.Dims

	// the colormap that we view
	Map *colormap.Map
}

func (cv *ColorMapView) OnInit() {
	cv.HandleColorMapEvents()
	// todo: style
}

// SetColorMap sets the color map and triggers a display update
func (cv *ColorMapView) SetColorMap(cmap *colormap.Map) {
	cv.Map = cmap
	cv.SetNeedsRender()
}

// SetColorMapAction sets the color map and triggers a display update
// and signals the ColorMapSig signal
func (cv *ColorMapView) SetColorMapAction(cmap *colormap.Map) {
	cv.Map = cmap
	cv.SendChange()
	cv.SetNeedsRender()
}

// ChooseColorMap pulls up a chooser to select a color map
func (cv *ColorMapView) ChooseColorMap() {
	sl := colormap.AvailMapsList()
	cur := ""
	if cv.Map != nil {
		cur = cv.Map.Name
	}
	SliceViewSelectDialog(cv, DlgOpts{Title: "Select a ColorMap", Prompt: "choose color map to use from among available list"}, &sl, cur, nil, func(dlg *gi.Dialog) {
		if !dlg.Accepted {
			return
		}
		si := dlg.Data.(int)
		if si >= 0 {
			nmap, ok := colormap.AvailMaps[sl[si]]
			if ok {
				cv.SetColorMapAction(nmap)
			}
		}
	}).Run()
}

func (cv *ColorMapView) HandleColorMapEvents() {
	cv.HandleWidgetEvents()
	cv.OnClick(func(e events.Event) {
		cv.ChooseColorMap()
	})

}

func (cv *ColorMapView) RenderColorMap(sc *gi.Scene) {
	if cv.Map == nil {
		cv.Map = colormap.StdMaps["ColdHot"]
	}
	rs := &sc.RenderState
	rs.Lock()
	pc := &rs.Paint

	pos := cv.LayState.Alloc.Pos
	sz := cv.LayState.Alloc.Size
	pr := pos
	sr := sz
	sp := pr.Dim(cv.Orient)

	lsz := sz.Dim(cv.Orient)

	if cv.Map.Indexed {
		nc := len(cv.Map.Colors)
		inc := mat32.Ceil(lsz / float32(nc))
		sr.SetDim(cv.Orient, inc)
		for i := 0; i < nc; i++ {
			clr := cv.Map.MapIndex(i)
			p := float32(i) * inc
			pr.SetDim(cv.Orient, sp+p)
			pc.FillBoxColor(rs, pr, sr, clr)
		}
	} else {
		inc := mat32.Ceil(lsz / 100)
		if inc < 2 {
			inc = 2
		}
		sr.SetDim(cv.Orient, inc)
		for p := float32(0); p < lsz; p += inc {
			val := p / (lsz - 1)
			clr := cv.Map.Map(float64(val))
			pr.SetDim(cv.Orient, sp+p)
			pc.FillBoxColor(rs, pr, sr, clr)
		}
	}
	rs.Unlock()
}

func (cv *ColorMapView) Render(sc *gi.Scene) {
	if cv.PushBounds(sc) {
		cv.RenderColorMap(sc)
		cv.RenderChildren(sc)
		cv.PopBounds(sc)
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  ColorMapValue

// Value registers ColorMapValue as the viewer of ColorMapName
func (mn ColorMapName) Value() Value {
	vv := &ColorMapValue{}
	ki.InitNode(vv)
	return vv
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

func (vv *ColorMapValue) ConfigWidget(widg gi.Widget, sc *gi.Scene) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(vv.Widget, nil)
	})
	vv.UpdateWidget()
}

func (vv *ColorMapValue) HasButton() bool {
	return true
}

func (vv *ColorMapValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsInactive() {
		return
	}
	sl := colormap.AvailMapsList()
	cur := laser.ToString(vv.Value.Interface())
	desc, _ := vv.Desc()
	SliceViewSelectDialog(ctx, DlgOpts{Title: "Select a ColorMap", Prompt: desc}, &sl, cur, nil, func(dlg *gi.Dialog) {
		if !dlg.Accepted {
			si := dlg.Data.(int)
			if si >= 0 {
				vv.SetValue(sl[si])
				vv.UpdateWidget()
			}
		}
		if fun != nil {
			fun(dlg)
		}
	}).Run()
}
