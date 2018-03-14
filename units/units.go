// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units

import (
	"fmt"
	"github.com/rcoreilly/goki/ki"
	"golang.org/x/image/math/fixed"
	"log"
	"strings"
)

// borrows from golang.org/x/exp/shiny/unit/ but extends with full range of css-based viewport-dependent factors

// standard conversion factors -- Px = DPI-independent pixel instead of actual "dot" raw pixel
const (
	PxPerInch = 96.0
	DpPerInch = 160.0
	MmPerInch = 25.4
	CmPerInch = 2.54
	PtPerInch = 72.0
	PcPerInch = 6.0
)

type Unit int32

const (
	// percentage of surrounding contextual element
	Pct Unit = iota
	// font size of the root element -- fallback to 12pt by default
	Rem
	// font size of the element -- fallback to 12pt by default
	Em
	// x-height of the element's font -- fallback to 0.5em by default
	Ex
	// with of the '0' glyph in the element's font -- fallback to 0.5em by default
	Ch
	// 1% of the viewport's width
	Vw
	// 1% of the viewport's height
	Vh
	// 1% of the viewport's smaller dimension
	Vmin
	// 1% of the viewport's larger dimension
	Vmax
	// centimeters -- 1cm = 96px/2.54
	Cm
	// millimeters -- 1mm = 1/10th of cm
	Mm
	// quarter-millimeters -- 1q = 1/40th of cm
	Q
	// inches -- 1in = 2.54cm = 96px
	In
	// picas -- 1pc = 1/6th of 1in
	Pc
	// points -- 1pt = 1/72th of 1in
	Pt
	// pixels -- 1px = 1/96th of 1in -- these are NOT raw display pixels
	Px
	// density-independent pixels -- 1dp = 1/160th of 1in
	Dp
)

var UnitNames = [...]string{
	Pct:  "pct",
	Rem:  "rem",
	Em:   "em",
	Ex:   "ex",
	Ch:   "ch",
	Vw:   "vw",
	Vh:   "vh",
	Vmin: "vmin",
	Vmax: "vmax",
	Cm:   "cm",
	Mm:   "mm",
	Q:    "q",
	In:   "in",
	Pc:   "pc",
	Pt:   "pt",
	Px:   "px",
	Dp:   "dp",
}

// UnitContext specifies everything about the current context necessary for converting the number
// into specific display-dependent pixels
type UnitContext struct {
	// dots-per-inch of the display
	DPI float64
	// point size of font (in points)
	FontEm float64
	// x-height of font in points
	FontEx float64
	// ch-size of font in points
	FontCh float64
	// rem-size of font in points
	FontRem float64
	// viewport width in raw pixels
	VpW float64
	// viewport height in raw pixels
	VpH float64
	// surrounding contextual element in raw pixels
	El float64
}

func (uc *UnitContext) Defaults() {
	uc.DPI = PxPerInch // default
	uc.FontEm = 12.0
	uc.FontEx = 6.0
	uc.FontCh = 6.0
	uc.FontRem = 12.0
	uc.VpW = 800.0
	uc.VpH = 600.0
	uc.El = uc.VpW
}

// factor needed to convert given unit into raw pixels (dots in DPI)
func (uc *UnitContext) ToDotsFactor(un Unit) float64 {
	if uc.DPI == 0 {
		log.Printf("UnitContext was not initialized -- falling back on defaults\n")
		uc.Defaults()
	}
	switch un {
	case Pct:
		return uc.El
	case Em:
		return uc.DPI / (PtPerInch / uc.FontEm)
	case Ex:
		return uc.DPI / (PtPerInch / uc.FontEx)
	case Ch:
		return uc.DPI / (PtPerInch / uc.FontCh)
	case Rem:
		return uc.DPI / (PtPerInch / uc.FontRem)
	case Vw:
		return uc.VpW
	case Vh:
		return uc.VpH
	case Vmin:
		return ki.Min64(uc.VpW, uc.VpH)
	case Vmax:
		return ki.Max64(uc.VpW, uc.VpH)
	case Cm:
		return uc.DPI / CmPerInch
	case Mm:
		return uc.DPI / MmPerInch
	case Q:
		return uc.DPI / (4.0 * MmPerInch)
	case In:
		return uc.DPI
	case Pc:
		return uc.DPI / PcPerInch
	case Pt:
		return uc.DPI / PtPerInch
	case Px:
		return uc.DPI / PxPerInch
	case Dp:
		return uc.DPI / DpPerInch
	}
	return uc.DPI
}

// convert value in given units into raw display pixels (dots in DPI)
func (uc *UnitContext) ToDots(val float64, un Unit) float64 {
	return val * uc.ToDotsFactor(un)
}

// Value is a number and a unit.
type Value struct {
	Val float64
	Un  Unit
}

// Convert value to raw display pixels (dots in DPI)
func (v Value) ToDots(ctxt *UnitContext) float64 {
	return ctxt.ToDots(v.Val, v.Un)
}

// Convert value to raw display pixels (dots in DPI) in fixed-point 26.6 format for rendering
func (v Value) ToDotsFixed(ctxt *UnitContext) fixed.Int26_6 {
	return fixed.Int26_6(ctxt.ToDots(v.Val, v.Un))
}

// Convert converts value to the given units, given unit context
func (v Value) Convert(to Unit, ctxt *UnitContext) Value {
	return Value{v.ToDots(ctxt) / ctxt.ToDotsFactor(to), to}
}

// String implements the fmt.Stringer interface.
func (v Value) String() string {
	return fmt.Sprintf("%f%s", v.Val, UnitNames[v.Un])
}

// parse string into a value
func StringToValue(str string) Value {
	trstr := strings.TrimSpace(str)
	var numstr string
	var un Unit = Px // default to pixels
	for i, nm := range UnitNames {
		if idx := strings.LastIndex(trstr, nm); idx > 0 {
			numstr = trstr[:idx]
			un = Unit(i)
			break
		}
	}
	if len(numstr) == 0 { // no units
		numstr = trstr
	}
	var val float64
	fmt.Sscanf(strings.TrimSpace(numstr), "%g", &val)
	return Value{val, un}
}
