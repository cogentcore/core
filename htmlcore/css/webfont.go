// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"strconv"
	"strings"

	"cogentcore.org/core/text/fonts"
)

// SplitTopLevel splits the given string at the given separator where it is
// not in parentheses or quotes (as a list of sources with url()s).
func SplitTopLevel(s string, sep byte) []string {
	var parts []string
	for {
		before, after, found := CutTopLevel(s, sep)
		parts = append(parts, before)
		if !found {
			return parts
		}
		s = after
	}
}

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
