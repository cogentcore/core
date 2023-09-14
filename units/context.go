// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units

// Context specifies everything about the current context necessary for converting the number
// into specific display-dependent pixels
type Context struct {

	// DPI is dots-per-inch of the display
	DPI float32

	// FontEm is the point size of the font in raw dots (not points)
	FontEm float32

	// FontEx is the height x-height of font in points (size of 'x' glyph)
	FontEx float32

	// FontCh is the ch-size character size of font in points (width of '0' glyph)
	FontCh float32

	// FontRem is rem-size of font in points -- root Em size -- typically 12 point
	FontRem float32

	// Vw is viewport width in dots
	Vw float32

	// Vh is viewport height in dots
	Vh float32

	// Ew is width of element in dots
	Ew float32

	// Eh is height of element in dots
	Eh float32

	// Pw is width of parent in dots
	Pw float32

	// Ph is height of parent in dots
	Ph float32
}

// Defaults are generic defaults
func (uc *Context) Defaults() {
	uc.DPI = PxPerInch
	uc.FontEm = 12.0
	uc.FontEx = 6.0
	uc.FontCh = 6.0
	uc.FontRem = 12.0
	uc.Vw = 800.0
	uc.Vh = 600.0
	uc.Ew = uc.Vw
	uc.Eh = uc.Vh
	uc.Pw = uc.Vw
	uc.Ph = uc.Vh
}

// Set sets the context values to the given values
func (uc *Context) Set(em, ex, ch, rem, vw, vh, ew, eh, pw, ph float32) {
	uc.SetSizes(vw, vh, ew, eh, pw, ph)
	uc.SetFont(em, ex, ch, rem)
}

// SetSizes sets the context values for the non-font sizes
// to the given values; the values are ignored if they are zero.
func (uc *Context) SetSizes(vw, vh, ew, eh, pw, ph float32) {
	if vw != 0 {
		uc.Vw = vw
	}
	if vh != 0 {
		uc.Vh = vh
	}
	if ew != 0 {
		uc.Ew = ew
	}
	if eh != 0 {
		uc.Eh = eh
	}
	if pw != 0 {
		uc.Pw = pw
	}
	if ph != 0 {
		uc.Ph = ph
	}
}

// SetFont sets the context values for fonts: note these are already in raw
// DPI dots, not points or anything else
func (uc *Context) SetFont(em, ex, ch, rem float32) {
	uc.FontEm = em
	uc.FontEx = ex
	uc.FontCh = ch
	uc.FontRem = rem
}

// ToDotsFact returns factor needed to convert given unit into raw pixels (dots in DPI)
func (uc *Context) Dots(un Units) float32 {
	if uc.DPI == 0 {
		// log.Printf("gi/units Context was not initialized -- falling back on defaults\n")
		uc.Defaults()
	}
	switch un {
	case UnitEw:
		return 0.01 * uc.Ew
	case UnitEh:
		return 0.01 * uc.Eh
	case UnitPw:
		return 0.01 * uc.Pw
	case UnitPh:
		return 0.01 * uc.Ph
	case UnitEm:
		return uc.FontEm
	case UnitEx:
		return uc.FontEx
	case UnitCh:
		return uc.FontCh
	case UnitRem:
		return uc.FontRem
	case UnitVw:
		return 0.01 * uc.Vw
	case UnitVh:
		return 0.01 * uc.Vh
	case UnitVmin:
		return min(uc.Vw, uc.Vh)
	case UnitVmax:
		return max(uc.Vw, uc.Vh)
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
