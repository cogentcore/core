// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"

	"cogentcore.org/core/htmlcore/css"

	"cogentcore.org/core/system"
)

// MediaWidth is the width of the viewport in CSS px (dp) that @media rules
// are evaluated for, when it is set for the context; otherwise it is the
// width of the screen.
func (c *Context) mediaWidth() float32 {
	if c.MediaWidth > 0 {
		return c.MediaWidth
	}
	return ScreenMediaWidth()
}

// ScreenMediaWidth returns the width of the screen in CSS px (dp), for the
// @media rules of style sheets when [Context.MediaWidth] is not set, or
// 1024 if there is no screen. An app that reads HTML in another goroutine
// should set MediaWidth (as to this) in its main goroutine first, as the
// screens can change in it (as when a window is made).
func ScreenMediaWidth() float32 {
	if system.TheApp != nil && system.TheApp.NScreens() > 0 {
		sc := system.TheApp.Screen(0)
		if sc.LogicalDPI > 0 && sc.PixelSize.X > 0 {
			return float32(sc.PixelSize.X) * 160 / sc.LogicalDPI
		}
	}
	return 1024
}

// MediaMatches returns whether the given media query list (as of an
// @media rule) matches the viewport of the context, as for matchMedia
// in the scripts of a page.
func (c *Context) MediaMatches(query string) bool {
	return c.mediaMatches(query)
}

// mediaMatches returns whether the given media query list matches,
// recording its result for [Context.MediaChanged].
func (c *Context) mediaMatches(query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	m := c.mediaListMatches(query)
	if !c.checkingMedia {
		if c.mediaUses == nil {
			c.mediaUses = map[string]bool{}
		}
		c.mediaUses[query] = m
	}
	return m
}

// MediaChanged returns whether the HTML read with the context would be
// read differently for a viewport of the given width in CSS px (dp):
// whether a media query of its style sheets (as of an @media rule) or a
// number of columns estimated from the width (for column-width) has
// another result for it. An app that shows pages in a viewport whose
// width can change, as when its window is resized, can then read the
// page again with a new context (see [Context.NewPage]) with that
// [Context.MediaWidth], as browsers apply the rules of the new width.
// It must be called in the goroutine that styles the widgets of the page,
// once the page has been read.
func (c *Context) MediaChanged(width float32) bool {
	if width <= 0 {
		return false
	}
	saved := c.MediaWidth
	c.MediaWidth, c.checkingMedia = width, true
	defer func() { c.MediaWidth, c.checkingMedia = saved, false }()
	for q, m := range c.mediaUses {
		if c.mediaListMatches(q) != m {
			return true
		}
	}
	for n, count := range c.columnUses {
		if c.columnCount(n) != count {
			return true
		}
	}
	return false
}

// mediaListMatches returns whether the given (non-empty, lowercase)
// media query list matches.
func (c *Context) mediaListMatches(query string) bool {
	for q := range strings.SplitSeq(query, ",") {
		if c.mediaQueryMatches(strings.TrimSpace(q)) {
			return true
		}
	}
	return false
}

// mediaQueryMatches returns whether the given media query matches.
func (c *Context) mediaQueryMatches(q string) bool {
	not := false
	if rest, ok := strings.CutPrefix(q, "not "); ok {
		not, q = true, rest
	}
	q = strings.TrimPrefix(q, "only ")
	match := true
	for part := range strings.SplitSeq(q, " and ") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.HasPrefix(part, "(") { // media type
			switch part {
			case "all", "screen":
			default:
				match = false
			}
			continue
		}
		if !c.mediaFeatureMatches(strings.Trim(part, "()")) {
			match = false
		}
	}
	return match != not
}

// mediaFeatureMatches returns whether the given media feature matches,
// treating features that are not supported as matching.
func (c *Context) mediaFeatureMatches(f string) bool {
	name, value, _ := strings.Cut(f, ":")
	name, value = strings.TrimSpace(name), strings.TrimSpace(value)
	length := func() (float32, bool) {
		return css.CalcPx(value)
	}
	width := c.mediaWidth()
	height := width * 3 / 4
	if c.MediaHeight > 0 {
		height = c.MediaHeight
	}
	if strings.Contains(f, "height") || strings.Contains(f, "aspect-ratio") || strings.Contains(f, "orientation") {
		c.viewportHeightUsed.Store(true)
	}
	if !strings.Contains(f, ":") && strings.ContainsAny(f, "<>=") {
		return css.MediaRangeMatches(f, width, height)
	}
	// the device is taken to be the size of the viewport, as for the
	// screen of a phone (and the window of headless Chrome)
	name = strings.Replace(name, "device-", "", 1)
	switch name {
	case "min-width", "max-width", "min-height", "max-height":
		l, ok := length()
		if !ok {
			return false // as browsers do for a query that is not valid
		}
		v := width
		if strings.HasSuffix(name, "height") {
			v = height
		}
		if strings.HasPrefix(name, "min") {
			return v >= l
		}
		return v <= l
	case "prefers-color-scheme":
		return value == "light"
	case "prefers-reduced-motion":
		// CSS animations and transitions are not run, so pages show the
		// final states of what they animate (as the headings of github.com
		// that fade in, which were hidden)
		return value == "reduce" || value == ""
	case "prefers-contrast", "forced-colors":
		return value == "no-preference" || value == "none"
	case "orientation":
		return value == "landscape"
	case "hover", "any-hover":
		return value == "hover"
	case "pointer", "any-pointer":
		return value == "fine"
	}
	return true
}
