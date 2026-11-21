// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"strconv"
	"strings"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/fonts"
	"cogentcore.org/core/text/rich"
)

// FontSizes are the absolute font size keywords, in px.
var FontSizes = map[string]float32{
	"xx-small": 9, "x-small": 10, "small": 13, "medium": 16, "large": 18,
	"x-large": 24, "xx-large": 32, "xxx-large": 48,
}

// LineHeightNormal is the line height of the "normal" value, which
// is given by the metrics of the font in browsers; this is the value
// for the default Noto Sans font.
const LineHeightNormal = 1.362

// SetFontWeight sets the font weight.
func SetFontWeight(s, ps *styles.Style, v string) {
	w := 400
	switch v {
	case "normal":
	case "bold":
		w = 700
	case "bolder":
		w = 700
		if ps != nil && ps.Font.Weight >= rich.Bold {
			w = 900
		}
	case "lighter":
		w = 300
	default:
		n, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		w = n
	}
	s.Font.Weight = rich.Weights(min(max((w+50)/100-1, 0), 8))
}

// FontFamily returns the font family for the given CSS font-family list,
// the first generic family in it, or one for a common font name.
func FontFamily(v string) rich.Family {
	for f := range strings.SplitSeq(v, ",") {
		f = strings.ToLower(strings.Trim(strings.TrimSpace(f), `"'`))
		switch f {
		case "monospace", "courier", "courier new", "consolas", "menlo", "monaco", "roboto mono", "ui-monospace", "sfmono-regular":
			return rich.Monospace
		case "serif", "times", "times new roman", "georgia", "garamond", "ui-serif":
			return rich.Serif
		case "sans-serif", "arial", "helvetica", "helvetica neue", "verdana", "tahoma", "system-ui", "-apple-system", "segoe ui", "roboto", "noto sans", "ui-sans-serif":
			return rich.SansSerif
		case "cursive":
			return rich.Cursive
		case "fantasy":
			return rich.Fantasy
		}
	}
	return rich.SansSerif
}

// FontSizeKeywords are the font sizes of the absolute size keywords, in px.
var FontSizeKeywords = map[string]float32{"xx-small": 9, "x-small": 10, "small": 13, "medium": 16,
	"large": 18, "x-large": 24, "xx-large": 32, "xxx-large": 48}

// IsFontData returns whether the given data is a font that the text
// shapers read: TrueType, OpenType (and collections of them), or WOFF.
func IsFontData(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	switch string(data[:4]) {
	case "\x00\x01\x00\x00", "OTTO", "true", "ttcf", "wOFF":
		return true
	}
	return false
}

// FontSource returns the URL and format of the given source of the src of
// an @font-face rule: url(...) format(...), or "" for local(...).
func FontSource(s string) (url, format string) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(strings.ToLower(s), "url(") {
		return "", ""
	}
	end := strings.Index(s, ")")
	if end < 0 {
		return "", ""
	}
	url = strings.Trim(strings.TrimSpace(s[4:end]), `"'`)
	rest := strings.ToLower(s[end+1:])
	if _, after, ok := strings.Cut(rest, "format("); ok {
		f := after
		if before, _, ok := strings.Cut(f, ")"); ok {
			format = strings.Trim(strings.TrimSpace(before), `"'`)
		}
	}
	return url, format
}

// UnicodeRangeHas returns whether the given unicode-range includes the
// given rune.
func UnicodeRangeHas(urange string, r rune) bool {
	for part := range strings.SplitSeq(urange, ",") {
		part = strings.ToUpper(strings.TrimSpace(part))
		part = strings.TrimPrefix(part, "U+")
		lo, hi := part, part
		if a, b, ok := strings.Cut(part, "-"); ok {
			lo, hi = a, b
		} else if strings.Contains(part, "?") { // as U+4??
			lo, hi = strings.ReplaceAll(part, "?", "0"), strings.ReplaceAll(part, "?", "F")
		}
		l, err1 := strconv.ParseUint(lo, 16, 32)
		h, err2 := strconv.ParseUint(hi, 16, 32)
		if err1 == nil && err2 == nil && uint64(r) >= l && uint64(r) <= h {
			return true
		}
	}
	return false
}

// WebFontFamily returns the first of the families of the given value of
// font-family that is a web font that has been loaded, or "".
func WebFontFamily(v string) string {
	for f := range strings.SplitSeq(v, ",") {
		f = strings.Trim(strings.TrimSpace(f), `"'`)
		if f != "" && fonts.HasWebFont(f) {
			return f
		}
	}
	return ""
}
