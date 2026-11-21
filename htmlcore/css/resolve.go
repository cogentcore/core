// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"strconv"
	"strings"

	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
)

// PercentUnit returns whether the given unit is a percentage of a size of
// the parent or the element, which is not definite for a replaced element.
func PercentUnit(u units.Units) bool {
	return u == units.UnitPw || u == units.UnitPh || u == units.UnitEw || u == units.UnitEh
}

// UnitlessCalc returns the value of the given calc() expression of numbers
// only (no lengths or percentages), and whether it is one.
func UnitlessCalc(v string) (float32, bool) {
	inner, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(v)), "calc(")
	if !ok || strings.ContainsAny(inner, "abcdefghijklmnopqrstuvwxyz%") {
		return 0, false
	}
	return CalcPx(v)
}

// ZeroLength returns whether the given CSS length is zero (as 0, 0px, 0%).
func ZeroLength(v string) bool {
	v = strings.TrimSpace(v)
	v = strings.TrimRight(v, "abcdefghijklmnopqrstuvwxyz%")
	f, err := strconv.ParseFloat(v, 32)
	return err == nil && f == 0
}

// SideDots returns the given sides in dots, using the given function.
func SideDots(v *sides.Values, dots func(v *units.Value) float32) sides.Floats {
	return sides.NewFloats(dots(&v.Top), dots(&v.Right), dots(&v.Bottom), dots(&v.Left))
}
