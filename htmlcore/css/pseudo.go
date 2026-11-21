// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"strconv"
	"strings"
)

// PseudoSelector returns the selector of the elements of the given
// selector of a ::before or ::after pseudo-element (also :before and
// :after), and which of them it is, or false if it is not one.
func PseudoSelector(sel string) (base, kind string, ok bool) {
	lsel := strings.ToLower(sel)
	for _, k := range []string{"before", "after"} {
		for _, p := range []string{"::" + k, ":" + k} {
			if strings.HasSuffix(lsel, p) {
				base = strings.TrimSpace(sel[:len(sel)-len(p)])
				if base == "" || strings.HasSuffix(base, " ") || strings.HasSuffix(base, ">") {
					base += "*"
				}
				return base, k, true
			}
		}
	}
	return "", "", false
}

// EmLength returns the given CSS length in em, with px for 16px to the
// em (and rem as em), or 0 if it is not one of these.
func EmLength(v string) float64 {
	v = strings.TrimSpace(strings.ToLower(v))
	for _, u := range []struct {
		suffix string
		scale  float64
	}{{"rem", 1}, {"em", 1}, {"px", 1.0 / 16}} {
		if num, ok := strings.CutSuffix(v, u.suffix); ok {
			f, err := strconv.ParseFloat(num, 64)
			if err != nil {
				return 0
			}
			return f * u.scale
		}
	}
	return 0
}

// EmSpaces returns a string of space characters (which are not collapsed
// or trimmed) about as wide as the given number of em.
func EmSpaces(em float64) string {
	var sb strings.Builder
	for _, sp := range []struct {
		r rune
		w float64
	}{{'\u2003', 1}, {'\u2002', 0.5}, {'\u2004', 1.0 / 3}, {'\u2005', 0.25}, {'\u2006', 1.0 / 6}, {'\u200A', 0.1}} {
		for em >= sp.w-0.04 && sb.Len() < 64 {
			sb.WriteRune(sp.r)
			em -= sp.w
		}
	}
	return sb.String()
}
