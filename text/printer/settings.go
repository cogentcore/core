// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate core generate -add-types

// package printer provides standard printer settings.
package printer

import (
	"path/filepath"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
)

// Settings provides the default printer settings.
// This is initialized by the system.App because it depends on
// the locale for initialization.
var Settings SettingsData

// Settings has standard printer settings.
type SettingsData struct {
	// PageSize specifies a standard page size, or Custom.
	PageSize PageSizes

	// Units are the units in which the page size is specified.
	// Will automatically be set if PageSize != Custom.
	Units units.Units

	// Size is the page size in given units.
	// Will automatically be set if PageSize != Custom.
	Size math32.Vector2

	// Margins specify the page margins in the size units.
	Margins sides.Floats `display:"inline"`
}

func (ps *SettingsData) Defaults() {
	ps.PageSize = DefaultPageSizeForRegion(system.TheApp.SystemLocale().Region())
	switch ps.Units {
	case units.UnitMm:
		ps.Margins.Set(25) // basically one inch
	case units.UnitPt:
		ps.Margins.Set(72)
	case units.UnitPx:
		ps.Margins.Set(24)
	}
	ps.Update()
}

func (ps *SettingsData) Update() {
	if ps.PageSize != Custom {
		pU := ps.Units
		ps.Units, ps.Size = ps.PageSize.Size()
		if pU != ps.Units {
			uc := units.NewContext()
			sc := uc.Convert(1, pU, ps.Units)
			ps.Margins = ps.Margins.MulScalar(sc)
		}
	}
}

func (ps *SettingsData) Apply() {
	ps.Update()
}

func (ps *SettingsData) Label() string {
	return "Printer"
}

func (ps *SettingsData) Filename() string {
	return filepath.Join(system.TheApp.CogentCoreDataDir(), "printer-settings.toml")
}

func (ps *SettingsData) MakeToolbar(p *tree.Plan) {
}

// ToDots returns the measurement values in rendering dots (actual pixels)
// based on the given units context.
// size = page size; body = content area inside margins
func (ps *SettingsData) ToDots(un *units.Context) (size, body math32.Vector2, margins sides.Floats) {
	sc := un.ToDots(1, ps.Units)
	size = ps.Size.MulScalar(sc).Floor()
	margins = ps.Margins.MulScalar(sc)
	body.X = size.X - (margins.Left + margins.Right)
	body.Y = size.Y - (margins.Top + margins.Bottom)
	return
}
