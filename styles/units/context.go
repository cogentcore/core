// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units

import (
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/math32"
)

// Context specifies everything about the current context necessary for converting the number
// into specific display-dependent pixels
type Context struct {

	// DPI is dots-per-inch of the display
	DPI float32

	// FontEm is the size of the font of the element in raw dots (not points)
	FontEm float32

	// FontEx is the height x-height of font in points (size of 'x' glyph)
	FontEx float32

	// FontCh is the ch-size character size of font in points (width of '0' glyph)
	FontCh float32

	// FontRem is the size of the font of the root element in raw dots (not points)
	FontRem float32

	// Vpw is viewport width in dots
	Vpw float32

	// Vph is viewport height in dots
	Vph float32

	// Elw is width of element in dots
	Elw float32

	// Elh is height of element in dots
	Elh float32

	// Paw is width of parent in dots
	Paw float32

	// Pah is height of parent in dots
	Pah float32
}

// Defaults are generic defaults
func (uc *Context) Defaults() {
	uc.DPI = DpPerInch
	uc.FontEm = 16
	uc.FontEx = 8
	uc.FontCh = 8
	uc.FontRem = 16
	uc.Vpw = 800
	uc.Vph = 600
	uc.Elw = uc.Vpw
	uc.Elh = uc.Vph
	uc.Paw = uc.Vpw
	uc.Pah = uc.Vph
}

func (uc *Context) String() string {
	return reflectx.StringJSON(uc)
}

// SetSizes sets the context values for the non-font sizes
// to the given values; the values are ignored if they are zero.
// returns true if any are different.
func (uc *Context) SetSizes(vw, vh, ew, eh, pw, ph float32) bool {
	diff := false
	if vw != 0 {
		if uc.Vpw != vw {
			diff = true
		}
		uc.Vpw = vw
	}
	if vh != 0 {
		if uc.Vph != vh {
			diff = true
		}
		uc.Vph = vh
	}
	if ew != 0 {
		if uc.Elw != ew {
			diff = true
		}
		uc.Elw = ew
	}
	if eh != 0 {
		if uc.Elh != eh {
			diff = true
		}
		uc.Elh = eh
	}
	if pw != 0 {
		if uc.Paw != pw {
			diff = true
		}
		uc.Paw = pw
	}
	if ph != 0 {
		if uc.Pah != ph {
			diff = true
		}
		uc.Pah = ph
	}
	return diff
}

// SetFont sets the context values for font based on the em size,
// which is the nominal font height, in DPI dots.
// This uses standard conversion factors from em. It is too unreliable
// and complicated to get these values from the actual font itself.
func (uc *Context) SetFont(em float32) {
	if em == 0 {
		em = 16
	}
	uc.FontEm = em
	uc.FontEx = math32.Round(0.53 * em)
	uc.FontCh = math32.Round(0.45 * em)
	uc.FontRem = math32.Round(uc.Dp(16))
}

// ToDotsFact returns factor needed to convert given unit into raw pixels (dots in DPI)
func (uc *Context) Dots(un Units) float32 {
	if uc.DPI == 0 {
		// log.Printf("gi/units Context was not initialized -- falling back on defaults\n")
		uc.Defaults()
	}
	switch un {
	case UnitEw:
		return 0.01 * uc.Elw
	case UnitEh:
		return 0.01 * uc.Elh
	case UnitPw:
		return 0.01 * uc.Paw
	case UnitPh:
		return 0.01 * uc.Pah
	case UnitEm:
		return uc.FontEm
	case UnitEx:
		return uc.FontEx
	case UnitCh:
		return uc.FontCh
	case UnitRem:
		return uc.FontRem
	case UnitVw:
		return 0.01 * uc.Vpw
	case UnitVh:
		return 0.01 * uc.Vph
	case UnitVmin:
		return 0.01 * min(uc.Vpw, uc.Vph)
	case UnitVmax:
		return 0.01 * max(uc.Vpw, uc.Vph)
	case UnitCm:
		return uc.DPI / CmPerInch
	case UnitMm:
		return uc.DPI / MmPerInch
	case UnitQ:
		return uc.DPI / (4.0 * MmPerInch)
	case UnitIn:
		return uc.DPI
	case UnitPc:
		return uc.DPI / PcPerInch
	case UnitPt:
		return uc.DPI / PtPerInch
	case UnitPx:
		return uc.DPI / PxPerInch
	case UnitDp:
		return uc.DPI / DpPerInch
	case UnitDot:
		return 1.0
	}
	return uc.DPI
}

// ToDots converts value in given units into raw display pixels (dots in DPI)
func (uc *Context) ToDots(val float32, un Units) float32 {
	return val * uc.Dots(un)
}

// PxToDots just converts a value from pixels to dots
func (uc *Context) PxToDots(val float32) float32 {
	return val * uc.Dots(UnitPx)
}

// DotsToPx just converts a value from dots to pixels
func (uc *Context) DotsToPx(val float32) float32 {
	return val / uc.Dots(UnitPx)
}
