// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"math"
	"reflect"
	"sort"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// ColorMap maps a value onto a color by interpolating between a list of colors
// defining a spectrum, or optionally as an indexed list of colors.
type ColorMap struct {
	Name    string
	NoColor gist.Color   `desc:"color to display for invalid numbers (e.g., NaN)"`
	Colors  []gist.Color `desc:"list of colors to interpolate between"`
	Indexed bool         `desc:"if true, this map should be used as an indexed list instead of interpolating a normalized floating point value: requires caller to check this flag and pass int indexes instead of normalized values to MapIndex"`
}

// Map returns color for normalized value in range 0-1.  NaN returns NoColor
// which can be used to indicate missing values.
func (cm *ColorMap) Map(val float64) gist.Color {
	nc := len(cm.Colors)
	if nc < 2 {
		return gist.Color{}
	}
	if math.IsNaN(val) {
		return cm.NoColor
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

// MapIndex returns color for given index, for scale in Indexed mode.
// NoColor is returned for values out of range of available colors.
// It is responsibility of the caller to use this method instead of Map
// based on the Indexed flag.
func (cm *ColorMap) MapIndex(val int) gist.Color {
	nc := len(cm.Colors)
	if val < 0 || val > nc {
		return cm.NoColor
	}
	return cm.Colors[val]
}

// see https://matplotlib.org/tutorials/colors/colormap-manipulation.html
// for how to read out matplotlib scales -- still don't understand segmented ones!

// StdColorMaps is a list of standard color maps
var StdColorMaps = map[string]*ColorMap{
	"ColdHot":        {"ColdHot", gist.Color{200, 200, 200, 255}, []gist.Color{{0, 255, 255, 255}, {0, 0, 255, 255}, {127, 127, 127, 255}, {255, 0, 0, 255}, {255, 255, 0, 255}}, false},
	"Jet":            {"Jet", gist.Color{200, 200, 200, 255}, []gist.Color{{0, 0, 127, 255}, {0, 0, 255, 255}, {0, 127, 255, 255}, {0, 255, 255, 255}, {127, 255, 127, 255}, {255, 255, 0, 255}, {255, 127, 0, 255}, {255, 0, 0, 255}, {127, 0, 0, 255}}, false},
	"JetMuted":       {"JetMuted", gist.Color{200, 200, 200, 255}, []gist.Color{{25, 25, 153, 255}, {25, 102, 230, 255}, {0, 230, 230, 255}, {0, 179, 0, 255}, {230, 230, 0, 255}, {230, 102, 25, 255}, {153, 25, 25, 255}}, false},
	"Viridis":        {"Viridis", gist.Color{200, 200, 200, 255}, []gist.Color{{72, 33, 114, 255}, {67, 62, 133, 255}, {56, 87, 140, 255}, {45, 111, 142, 255}, {36, 133, 142, 255}, {30, 155, 138, 255}, {42, 176, 127, 255}, {81, 197, 105, 255}, {134, 212, 73, 255}, {194, 223, 35, 255}, {253, 231, 37, 255}}, false},
	"Plasma":         {"Plasma", gist.Color{200, 200, 200, 255}, []gist.Color{{61, 4, 155, 255}, {99, 0, 167, 255}, {133, 6, 166, 255}, {166, 32, 152, 255}, {192, 58, 131, 255}, {213, 84, 110, 255}, {231, 111, 90, 255}, {246, 141, 69, 255}, {253, 174, 50, 255}, {252, 210, 36, 255}, {240, 248, 33, 255}}, false},
	"Inferno":        {"Inferno", gist.Color{200, 200, 200, 255}, []gist.Color{{37, 12, 3, 255}, {19, 11, 52, 255}, {57, 9, 99, 255}, {95, 19, 110, 255}, {133, 33, 107, 255}, {169, 46, 94, 255}, {203, 65, 73, 255}, {230, 93, 47, 255}, {247, 131, 17, 255}, {252, 174, 19, 255}, {245, 219, 76, 255}, {252, 254, 164, 255}}, false},
	"BlueBlackRed":   {"BlueBlackRed", gist.Color{200, 200, 200, 255}, []gist.Color{{0, 0, 255, 255}, {76, 76, 76, 255}, {255, 0, 0, 255}}, false},
	"BlueGreyRed":    {"BlueGreyRed", gist.Color{200, 200, 200, 255}, []gist.Color{{0, 0, 255, 255}, {127, 127, 127, 255}, {255, 0, 0, 255}}, false},
	"BlueWhiteRed":   {"BlueWhiteRed", gist.Color{200, 200, 200, 255}, []gist.Color{{0, 0, 255, 255}, {230, 230, 230, 255}, {255, 0, 0, 255}}, false},
	"BlueGreenRed":   {"BlueGreenRed", gist.Color{200, 200, 200, 255}, []gist.Color{{0, 0, 255, 255}, {0, 230, 0, 255}, {255, 0, 0, 255}}, false},
	"Rainbow":        {"Rainbow", gist.Color{200, 200, 200, 255}, []gist.Color{{255, 0, 255, 255}, {0, 0, 255, 255}, {0, 255, 0, 255}, {255, 255, 0, 255}, {255, 0, 0, 255}}, false},
	"ROYGBIV":        {"ROYGBIV", gist.Color{200, 200, 200, 255}, []gist.Color{{255, 0, 255, 255}, {0, 0, 127, 255}, {0, 0, 255, 255}, {0, 255, 0, 255}, {255, 255, 0, 255}, {255, 0, 0, 255}}, false},
	"DarkLight":      {"DarkLight", gist.Color{200, 200, 200, 255}, []gist.Color{{0, 0, 0, 255}, {250, 250, 250, 255}}, false},
	"DarkLightDark":  {"DarkLightDark", gist.Color{200, 200, 200, 255}, []gist.Color{{0, 0, 0, 255}, {250, 250, 250, 255}, {0, 0, 0, 255}}, false},
	"LightDarkLight": {"DarkLightDark", gist.Color{200, 200, 200, 255}, []gist.Color{{250, 250, 250, 255}, {0, 0, 0, 255}, {250, 250, 250, 255}}, false},
}

// AvailColorMaps is the list of all available color maps
var AvailColorMaps = map[string]*ColorMap{}

func init() {
	for k, v := range StdColorMaps {
		AvailColorMaps[k] = v
	}
}

// AvailColorMapsList returns a sorted list of color map names, e.g., for choosers
func AvailColorMapsList() []string {
	sl := make([]string, len(AvailColorMaps))
	ctr := 0
	for k := range AvailColorMaps {
		sl[ctr] = k
		ctr++
	}
	sort.Strings(sl)
	return sl
}

// ColorMapName provides a gui chooser of maps in AvailColorMaps
type ColorMapName string

/////////////////////////////////////////////////////////////////////////////
//  ColorMapView

// ColorMapView is a widget that displays a ColorMap.
// Note that this is not a ValueView widget
type ColorMapView struct {
	gi.WidgetBase
	Orient      mat32.Dims `desc:"orientation along which to display the spectrum"`
	Map         *ColorMap  `desc:"the colormap that we view"`
	ColorMapSig ki.Signal  `json:"-" xml:"-" view:"-" desc:"signal for color map -- triggers when new color map is set via chooser"`
}

var KiT_ColorMapView = kit.Types.AddType(&ColorMapView{}, nil)

// AddNewColorMapView adds a new colorview to given parent node, with given name.
func AddNewColorMapView(parent ki.Ki, name string, cmap *ColorMap) *ColorMapView {
	cv := parent.AddNewChild(KiT_ColorMapView, name).(*ColorMapView)
	cv.Map = cmap
	return cv
}

func (cv *ColorMapView) Disconnect() {
	cv.WidgetBase.Disconnect()
	cv.ColorMapSig.DisconnectAll()
}

// SetColorMap sets the color map and triggers a display update
func (cv *ColorMapView) SetColorMap(cmap *ColorMap) {
	cv.Map = cmap
	cv.UpdateSig()
}

// SetColorMapAction sets the color map and triggers a display update
// and signals the ColorMapSig signal
func (cv *ColorMapView) SetColorMapAction(cmap *ColorMap) {
	cv.Map = cmap
	cv.ColorMapSig.Emit(cv.This(), 0, nil)
	cv.UpdateSig()
}

// ChooseColorMap pulls up a chooser to select a color map
func (cv *ColorMapView) ChooseColorMap() {
	sl := AvailColorMapsList()
	cur := ""
	if cv.Map != nil {
		cur = cv.Map.Name
	}
	SliceViewSelectDialog(cv.Viewport, &sl, cur, DlgOpts{Title: "Select a ColorMap", Prompt: "choose color map to use from among available list"}, nil,
		cv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
				si := SliceViewSelectDialogValue(ddlg)
				if si >= 0 {
					nmap, ok := AvailColorMaps[sl[si]]
					if ok {
						cv.SetColorMapAction(nmap)
					}
				}
			}
		})
}

// MouseEvent handles button MouseEvent
func (cv *ColorMapView) MouseEvent() {
	cv.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
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

func (cv *ColorMapView) ConnectEvents2D() {
	cv.MouseEvent()
	cv.HoverTooltipEvent()
}

func (cv *ColorMapView) RenderColorMap() {
	if cv.Map == nil {
		cv.Map = StdColorMaps["ColdHot"]
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

var KiT_ColorMapValueView = kit.Types.AddType(&ColorMapValueView{}, nil)

func (vv *ColorMapValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *ColorMapValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	txt := kit.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none, click to select)"
	}
	ac.SetText(txt)
}

func (vv *ColorMapValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Action)
	ac.SetProp("border-radius", units.NewPx(4))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_ColorMapValueView).(*ColorMapValueView)
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
	sl := AvailColorMapsList()
	cur := kit.ToString(vv.Value.Interface())
	desc, _ := vv.Tag("desc")
	SliceViewSelectDialog(vp, &sl, cur, DlgOpts{Title: "Select a ColorMap", Prompt: desc}, nil,
		vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
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
