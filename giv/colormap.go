// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"

	"goki.dev/colors/colormap"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/gist"
	"goki.dev/goosi"
	"goki.dev/goosi/mouse"
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

	// [view: -] signal for color map -- triggers when new color map is set via chooser
	ColorMapSig ki.Signal `json:"-" xml:"-" view:"-" desc:"signal for color map -- triggers when new color map is set via chooser"`
}

func (cv *ColorMapView) Disconnect() {
	cv.WidgetBase.Disconnect()
	cv.ColorMapSig.DisconnectAll()
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
	cv.ColorMapSig.Emit(cv.This(), 0, nil)
	cv.UpdateSig()
}

// ChooseColorMap pulls up a chooser to select a color map
func (cv *ColorMapView) ChooseColorMap() {
	sl := colormap.AvailMapsList()
	cur := ""
	if cv.Map != nil {
		cur = cv.Map.Name
	}
	SliceViewSelectDialog(cv.Viewport, &sl, cur, DlgOpts{Title: "Select a ColorMap", Prompt: "choose color map to use from among available list"}, nil,
		cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.TypeDialog).(*gi.Dialog)
				si := SliceViewSelectDialogValue(ddlg)
				if si >= 0 {
					nmap, ok := colormap.AvailMaps[sl[si]]
					if ok {
						cv.SetColorMapAction(nmap)
					}
				}
			}
		})
}

// MouseEvent handles button MouseEvent
func (cv *ColorMapView) MouseEvent() {
	cv.ConnectEvent(goosi.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		cvv := recv.(*ColorMapView)
		if me.Button == mouse.Left {
			switch me.Action {
			case mouse.DoubleClick: // we just count as a regular click
				fallthrough
			case mouse.Press:
				me.SetProcessed()
				cvv.ChooseColorMap()
			}
		}
	})
}

func (cv *ColorMapView) ConnectEvents() {
	cv.MouseEvent()
	cv.HoverTooltipEvent()
}

func (cv *ColorMapView) RenderColorMap() {
	if cv.Map == nil {
		cv.Map = colormap.StdMaps["ColdHot"]
	}
	rs := cv.Render()
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

func (cv *ColorMapView) Render() {
	if cv.FullReRenderIfNeeded() {
		return
	}
	if cv.PushBounds() {
		cv.This().(gi.Node2D).ConnectEvents()
		cv.RenderColorMap()
		cv.RenderChildren()
		cv.PopBounds()
	} else {
		cv.DisconnectAllEvents(gi.RegPri)
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

// ColorMapValueView presents an action for displaying a ColorMapName and selecting
// meshes from a ChooserDialog
type ColorMapValueView struct {
	ValueViewBase
}

func (vv *ColorMapValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.TypeAction
	return vv.WidgetTyp
}

func (vv *ColorMapValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none, click to select)"
	}
	ac.SetText(txt)
}

func (vv *ColorMapValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Action)
	ac.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		s.Border.Radius = gist.BorderRadiusFull
	})
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(TypeColorMapValueView).(*ColorMapValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.Viewport, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *ColorMapValueView) HasAction() bool {
	return true
}

func (vv *ColorMapValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	sl := colormap.AvailMapsList()
	cur := laser.ToString(vv.Value.Interface())
	desc, _ := vv.Tag("desc")
	SliceViewSelectDialog(vp, &sl, cur, DlgOpts{Title: "Select a ColorMap", Prompt: desc}, nil,
		vv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.TypeDialog).(*gi.Dialog)
				si := SliceViewSelectDialogValue(ddlg)
				if si >= 0 {
					vv.SetValue(sl[si])
					vv.UpdateWidget()
				}
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})
}
