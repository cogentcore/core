// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"regexp"
	"strings"
)

// Origin is the origin of a CSS declaration, which orders the cascade.
type Origin int

// PropertyName returns the name of the property of a declaration with the
// given name: in lower case, and the standard one of a legacy alias of it
// (see [legacyAliases]).
func PropertyName(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	if a, ok := legacyAliases[p]; ok {
		return a
	}
	return p
}

// legacyAliases are the old names of properties that browsers still
// support as aliases of the standard ones, by their standard names (as the
// grid-column-gap of the grids of nytimes.com, without which their columns
// had no gaps between them).
var legacyAliases = map[string]string{
	"grid-gap": "gap", "grid-row-gap": "row-gap", "grid-column-gap": "column-gap",
}

// otherVendorValue matches a keyword of the vendor prefix of another
// browser engine (as -moz-available), which is invalid in browsers of
// WebKit and Blink, as Cogent Core is.
var otherVendorValue = regexp.MustCompile(`(^|[\s,(])-(moz|ms|o)-[a-z]`)

// ValidDeclaration returns whether a declaration of the given property
// and value is valid, as in browsers, which drop the declarations that
// are not when they are parsed, so that the declarations before them of
// the same property apply (as height: 100vh before height:
// -webkit-fill-available, of the captcha frames of DataDome for Safari,
// which were of the height of no content). A size of
// -webkit-fill-available is invalid here too, as it is not laid out, and
// the declaration before it is meant as a fallback for it. Custom
// properties and values with variables are only known when they are used.
func ValidDeclaration(prop, value string) bool {
	if strings.HasPrefix(prop, "--") || strings.Contains(value, "var(") {
		return true
	}
	if otherVendorValue.MatchString(strings.ToLower(value)) {
		return false
	}
	switch prop {
	case "width", "height", "min-width", "min-height", "max-width", "max-height", "inline-size", "block-size":
		if strings.Contains(strings.ToLower(value), "-webkit-fill-available") {
			return false
		}
	}
	return true
}

// LegacyDisplay returns the display for the given value of the display
// property, which is that of the prefixed legacy values: -webkit-box (as
// with -webkit-line-clamp, in a column) is a block, and -webkit-flex flex;
// unset and initial are inline.
func LegacyDisplay(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "-webkit-box", "-moz-box":
		return "block"
	case "-webkit-inline-box", "-moz-inline-box":
		return "inline-block"
	case "-webkit-flex":
		return "flex"
	case "-webkit-inline-flex":
		return "inline-flex"
	case "unset", "initial":
		// display is not inherited, so these are its initial value (as
		// display: unset of the widgets of the masthead of nytimes.com, in
		// a flex row, which had no box of its own, so that its ticker was
		// as wide as the row, over the logo)
		return "inline"
	}
	return v
}

// FontShorthandValid returns whether the given value of the font shorthand
// is valid: a size (as 14px or medium, after any style, variant and
// weight) and a family after it, or a CSS-wide keyword or system font.
func FontShorthandValid(v string) bool {
	fs := strings.Fields(strings.ToLower(v))
	if len(fs) == 1 {
		switch fs[0] {
		case "inherit", "initial", "unset", "revert", "revert-layer", "caption", "icon", "menu",
			"message-box", "small-caption", "status-bar":
			return true
		}
	}
	for i, f := range fs {
		if len(f) > 0 && (f[0] >= '0' && f[0] <= '9' || f[0] == '.') || FontSizes[f] > 0 ||
			strings.HasPrefix(f, "calc(") || strings.HasPrefix(f, "clamp(") || strings.HasPrefix(f, "min(") || strings.HasPrefix(f, "max(") {
			return i+1 < len(fs) // (the family after the size)
		}
	}
	return false
}

const (
	// OriginUserAgent is for the user agent (default) styles.
	OriginUserAgent Origin = iota

	// OriginHint is for the presentational attributes of
	// elements (width, bgcolor etc), which come before author styles.
	OriginHint

	// OriginAuthor is for the styles of the page.
	OriginAuthor

	// OriginInline is for the style attribute of an element.
	OriginInline
)
