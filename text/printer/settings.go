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

func init() {
	Settings.Defaults()
}

// Settings provides the default printer settings.
var Settings SettingsData

// Settings has standard printer settings.
type SettingsData struct {
	// PageSize specifies a standard page size, or Custom.
	PageSize PageSizes

	// Units are the units in which size is specified.
	// Will automatically be set if PageSize != Custom.
	Units units.Units

	// Size is the size in given units.
	// Will automatically be set if PageSize != Custom.
	Size math32.Vector2

	// Margins specify the page margins in the size units.
	Margins sides.Floats `display:"inline"`
}

func (ps *SettingsData) Defaults() {
	ps.PageSize = DefaultPageSizeForRegion(system.TheApp.SystemLocale().Region())
	ps.Margins.Set(25) // basically one inch
	ps.Update()
}

func (ps *SettingsData) Update() {
	if ps.PageSize != Custom {
		ps.Units, ps.Size = ps.PageSize.Size()
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
	size = ps.Size.MulScalar(sc)
	margins = ps.Margins.MulScalar(sc)
	body.X = size.X - (margins.Left + margins.Right)
	body.Y = size.Y - (margins.Top + margins.Bottom)
	return
}
