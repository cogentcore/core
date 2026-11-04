// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strconv"
	"strings"

	"cogentcore.org/core/system"
	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
)

// parseCSS parses the given style sheet. If it does not parse as a whole,
// as with constructs that the parser does not support, each of its top-level
// rules is parsed on its own, so that only the rules that do not parse are
// left out.
func parseCSS(style string) *css.Stylesheet {
	if ss, err := parser.Parse(style); err == nil {
		return ss
	}
	ss := css.NewStylesheet()
	for _, block := range topLevelBlocks(style) {
		bs, err := parser.Parse(block)
		if err != nil {
			continue
		}
		ss.Rules = append(ss.Rules, bs.Rules...)
	}
	return ss
}

// topLevelBlocks splits the given style sheet into its top-level rules,
// at the closing braces (or semicolons of at-rules without blocks) that
// are not nested, in strings, or in comments.
func topLevelBlocks(style string) []string {
	var blocks []string
	depth := 0
	start := 0
	var quote byte
	for i := 0; i < len(style); i++ {
		ch := style[i]
		switch {
		case quote != 0:
			if ch == '\\' {
				i++
			} else if ch == quote {
				quote = 0
			}
		case ch == '"' || ch == '\'':
			quote = ch
		case ch == '/' && i+1 < len(style) && style[i+1] == '*':
			end := strings.Index(style[i+2:], "*/")
			if end < 0 {
				i = len(style)
			} else {
				i += end + 3
			}
		case ch == '{':
			depth++
		case ch == '}':
			depth--
			if depth <= 0 {
				blocks = append(blocks, style[start:i+1])
				start = i + 1
				depth = 0
			}
		case ch == ';' && depth == 0:
			blocks = append(blocks, style[start:i+1])
			start = i + 1
		}
	}
	return blocks
}

// MediaWidth is the width of the viewport in CSS px (dp) that @media rules
// are evaluated for, when it is set for the context; otherwise it is the
// width of the screen.
func (c *Context) mediaWidth() float32 {
	if c.MediaWidth > 0 {
		return c.MediaWidth
	}
	if system.TheApp != nil && system.TheApp.NScreens() > 0 {
		sc := system.TheApp.Screen(0)
		if sc.LogicalDPI > 0 && sc.PixelSize.X > 0 {
			return float32(sc.PixelSize.X) * 160 / sc.LogicalDPI
		}
	}
	return 1024
}

// mediaMatches returns whether the given media query list matches.
func (c *Context) mediaMatches(query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	for _, q := range strings.Split(query, ",") {
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
	for _, part := range strings.Split(q, " and ") {
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
		v := strings.TrimSuffix(value, "px")
		scale := float32(1)
		if strings.HasSuffix(value, "em") {
			v = strings.TrimSuffix(strings.TrimSuffix(value, "rem"), "em")
			scale = 16
		}
		n, err := strconv.ParseFloat(v, 32)
		return float32(n) * scale, err == nil
	}
	width := c.mediaWidth()
	height := width * 3 / 4
	switch name {
	case "min-width":
		if l, ok := length(); ok {
			return width >= l
		}
	case "max-width":
		if l, ok := length(); ok {
			return width <= l
		}
	case "min-height":
		if l, ok := length(); ok {
			return height >= l
		}
	case "max-height":
		if l, ok := length(); ok {
			return height <= l
		}
	case "prefers-color-scheme":
		return value == "light"
	case "prefers-reduced-motion", "prefers-contrast", "forced-colors":
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
