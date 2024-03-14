// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"log/slog"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/colormap"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
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
	ValueBase[*gi.Frame]
}

func (v *ColorMapValue) Config() {
	v.Widget.HandleClickOnEnterSpace()
	ConfigDialogWidget(v, false)
	v.Widget.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Hoverable, abilities.Clickable, abilities.Focusable)
		s.Cursor = cursors.Pointer
		s.Border.Radius = styles.BorderRadiusMedium

		s.Grow.Set(0, 0)
		s.Min.Set(units.Em(10), units.Em(1.5))

		cmn, ok := laser.NonPtrValue(v.Value).Interface().(ColorMapName)
		if !ok || cmn == "" {
			s.Background = colors.C(colors.Scheme.OutlineVariant)
			return
		}
		cm, ok := colormap.AvailMaps[string(cmn)]
		if !ok {
			slog.Error("got invalid color map name", "name", cmn)
			s.Background = colors.C(colors.Scheme.OutlineVariant)
			return
		}
		g := gradient.NewLinear()
		for i := float32(0); i < 1; i += 0.01 {
			gc := cm.Map(i)
			g.AddStop(gc, i)
		}
		s.Background = g
	})
}

func (v *ColorMapValue) Update() {
	v.Widget.ApplyStyleTree()
	v.Widget.NeedsRender()
}

// TODO(dtl): Select a color map
func (v *ColorMapValue) ConfigDialog(d *gi.Body) (bool, func()) {
	sl := colormap.AvailMapsList()
	cur := laser.ToString(v.Value.Interface())
	si := 0
	NewSliceView(d).SetSlice(&sl).SetSelVal(cur).BindSelect(&si)
	return true, func() {
		if si >= 0 {
			v.SetValue(sl[si])
			v.Update()
		}
	}
}
