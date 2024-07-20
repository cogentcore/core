// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"log/slog"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/colormap"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// ColorMapName represents the name of a [colormap.Map],
// which can be edited using a [ColorMapButton].
type ColorMapName string

func (cm ColorMapName) Value() Value { return NewColorMapButton() }

// ColorMapButton displays a [colormap.Map] and can be clicked on
// to display a dialog for selecting different color map options.
// It represents a [ColorMapName] value.
type ColorMapButton struct {
	Button
	MapName string
}

func (cm *ColorMapButton) WidgetValue() any { return &cm.MapName }

func (cm *ColorMapButton) Init() {
	cm.Button.Init()
	cm.Styler(func(s *styles.Style) {
		s.Padding.Zero()
		s.Min.Set(units.Em(10), units.Em(2))

		if cm.MapName == "" {
			s.Background = colors.Scheme.OutlineVariant
			return
		}
		cm, ok := colormap.AvailableMaps[cm.MapName]
		if !ok {
			slog.Error("got invalid color map name", "name", cm.Name)
			s.Background = colors.Scheme.OutlineVariant
			return
		}
		g := gradient.NewLinear()
		for i := float32(0); i < 1; i += 0.01 {
			gc := cm.Map(i)
			g.AddStop(gc, i)
		}
		s.Background = g
	})

	InitValueButton(cm, false, func(d *Body) {
		d.SetTitle("Select a color map")
		sl := colormap.AvailableMapsList()
		si := 0
		ls := NewList(d).SetSlice(&sl).SetSelectedValue(cm.MapName).BindSelect(&si)
		ls.OnChange(func(e events.Event) {
			cm.MapName = sl[si]
		})
	})
}
