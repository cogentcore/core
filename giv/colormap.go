// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"log/slog"

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

// ColorMapName represents the name of a color map, which can be edited using a [ColorMapValue].
type ColorMapName string

func (cmn ColorMapName) Value() Value {
	return &ColorMapValue{}
}

// ColorMapValue displays a color map spectrum and can be clicked on
// to display a dialog for selecting different color map options.
// It represents a [ColorMapName] value.
type ColorMapValue struct {
	ValueBase

	// Dim is the dimension on which to display the color map spectrum
	Dim mat32.Dims
}

func (vv *ColorMapValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.FrameType
	return vv.WidgetTyp
}

func (vv *ColorMapValue) UpdateWidget() {}

func (vv *ColorMapValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	fr := vv.Widget.(*gi.Frame)
	fr.Config()
	fr.OnClick(func(e events.Event) {
		if !vv.IsReadOnly() {
			vv.OpenDialog(vv.Widget, nil)
		}
	})
	fr.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable, abilities.Pressable)
		s.Cursor = cursors.Pointer
		s.Border.Radius = styles.BorderRadiusExtraSmall

		cmn, ok := laser.NonPtrValue(vv.Value).Interface().(ColorMapName)
		if !ok || cmn == "" {
			return
		}
		cm, ok := colormap.AvailMaps[string(cmn)]
		if !ok {
			slog.Error("got invalid color map name", cmn)
			return
		}
		s.BackgroundColor.Gradient = colors.LinearGradient()
		for i := float32(0); i < 1; i += 0.01 {
			gc := cm.Map(i)
			s.BackgroundColor.Gradient.AddStop(gc, i, 1)
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
