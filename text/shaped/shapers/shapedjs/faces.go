// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedjs

import (
	"strconv"
	"strings"

	"cogentcore.org/core/text/rich"
)

// webFace is a web font face that has finished loading.
type webFace struct {
	// family is the family name, without quotes.
	family string

	// italic is whether the face is italic (or oblique).
	italic bool

	// ranges are the inclusive ranges of the runes of the face,
	// or nil for all runes.
	ranges [][2]rune
}

// parseUnicodeRange returns the inclusive rune ranges of a CSS unicode-range,
// such as "U+0-FF, U+131, U+4??", or nil (all runes) if it cannot parse it.
func parseUnicodeRange(s string) [][2]rune {
	var ranges [][2]rune
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if len(part) < 3 || (part[0] != 'U' && part[0] != 'u') || part[1] != '+' {
			return nil
		}
		part = part[2:]

		// A range is lo-hi; wildcards (?) are the lowest and highest digits.
		lo, hi, ok := strings.Cut(part, "-")
		if !ok {
			lo, hi = strings.ReplaceAll(part, "?", "0"), strings.ReplaceAll(part, "?", "F")
		}
		l, err1 := strconv.ParseUint(lo, 16, 32)
		h, err2 := strconv.ParseUint(hi, 16, 32)
		if err1 != nil || err2 != nil {
			return nil
		}
		ranges = append(ranges, [2]rune{rune(l), rune(h)})
	}
	return ranges
}

// has returns whether the face has any of the runes.
func (wf *webFace) has(rs []rune) bool {
	if wf.ranges == nil {
		return len(rs) > 0
	}
	for _, r := range rs {
		for _, rg := range wf.ranges {
			if r >= rg[0] && r <= rg[1] {
				return true
			}
		}
	}
	return false
}

// usedBy returns whether the face is one that texts in the given font
// family list and slant are measured with: a regular face is also used for
// italic texts without an italic face.
func (wf *webFace) usedBy(families string, italic bool) bool {
	if wf.italic && !italic {
		return false
	}
	for _, f := range strings.Split(families, ",") {
		if strings.Trim(f, `"' `) == wf.family {
			return true
		}
	}
	return false
}

// textUsesFaces returns whether the text has runes that are measured with
// any of the given faces.
func textUsesFaces(tx rich.Text, customFont rich.FontName, faces []webFace) bool {
	for si := range tx {
		sty, rs := tx.Span(si)
		if sty.IsMath() {
			continue
		}
		families := string(customFont)
		if sty.Family != rich.Custom {
			families = sty.FontFamily(&rich.Settings)
		}
		for i := range faces {
			if faces[i].usedBy(families, sty.Slant == rich.Italic) && faces[i].has(rs) {
				return true
			}
		}
	}
	return false
}
