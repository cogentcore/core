// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/colors/colormap"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
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
// Note that this is not a ValueView widget
type ColorMapView struct {
	gi.WidgetBase

	// orientation along which to display the spectrum
	Orient mat32.Dims `desc:"orientation along which to display the spectrum"`

	// the colormap that we view
	Map *colormap.Map `desc:"the colormap that we view"`
}

func (cv *ColorMapView) OnInit() {
	cv.ColorMapHandlers()
	// todo: style
}

// SetColorMap sets the color map and triggers a display update
func (cv *ColorMapView) SetColorMap(cmap *colormap.Map) {
	cv.Map = cmap
	cv.UpdateSig()
}

// SetColorMapAction sets the color map and triggers a display update
// and signals the ColorMapSig signal
func (cv *ColorMapView) SetColorMapAction(cmap *colormap.Map) {
	cv.Map = cmap
	cv.SendChange()
	cv.UpdateSig()
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
		// todo: use data for this!
		// si := SliceViewSelectDialogValue(ddlg)
		// if si >= 0 {
		// 		nmap, ok := colormap.AvailMaps[sl[si]]
		// 			if ok {
		// 				cv.SetColorMapAction(nmap)
		// 			}
	})
}

func (cv *ColorMapView) ColorMapHandlers() {
	cv.WidgetHandlers()
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
//  ColorMapValueView

// ValueView registers ColorMapValueView as the viewer of ColorMapName
func (mn ColorMapName) ValueView() ValueView {
	vv := &ColorMapValueView{}
	ki.InitNode(vv)
	return vv
}

// ColorMapValueView presents an button for displaying a ColorMapName and selecting
// meshes from a ChooserDialog
type ColorMapValueView struct {
	ValueViewBase
}

func (vv *ColorMapValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *ColorMapValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none, click to select)"
	}
	ac.SetText(txt)
}

func (vv *ColorMapValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Button)
	ac.AddStyles(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusFull
	})
	ac.OnClick(func(e events.Event) {
		vv.OpenDialog(vv.Widget, nil)
	})
	vv.UpdateWidget()
}

func (vv *ColorMapValueView) HasButton() bool {
	return true
}

func (vv *ColorMapValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsInactive() {
		return
	}
	sl := colormap.AvailMapsList()
	cur := laser.ToString(vv.Value.Interface())
	desc, _ := vv.Tag("desc")
	SliceViewSelectDialog(ctx, DlgOpts{Title: "Select a ColorMap", Prompt: desc}, &sl, cur, nil, func(dlg *gi.Dialog) {
		if !dlg.Accepted {
			return
		}
		// todo: use data
		// si := SliceViewSelectDialogValue(ddlg)
		// if si >= 0 {
		// 	vv.SetValue(sl[si])
		// 	vv.UpdateWidget()
		// }
		//
		if fun != nil {
			fun(dlg)
		}
	})
}
