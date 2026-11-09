// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/system"
	"golang.org/x/net/html"
)

// The responsive images of HTML: the source of an img is chosen from the
// source elements of its picture (by their media and type) and from the
// candidates of a srcset (by their density, or their width for the sizes
// of the image), as in browsers, as the images of nytimes.com/wirecutter,
// which have a srcset and no src.

// imageSource returns the URL of the image to show for the given img
// element, and its density (the number of its pixels per CSS px), or ""
// if it has none.
func (c *Context) imageSource(n *html.Node) (src string, density float32) {
	sizes := GetAttr(n, "sizes")
	if p := n.Parent; p != nil && p.Type == html.ElementNode && p.Data == "picture" {
		for s := p.FirstChild; s != nil && s != n; s = s.NextSibling {
			if s.Type != html.ElementNode || s.Data != "source" || !imageTypeSupported(GetAttr(s, "type")) {
				continue
			}
			if m := GetAttr(s, "media"); m != "" && !c.mediaMatches(m) {
				continue
			}
			if HasAttr(s, "sizes") {
				sizes = GetAttr(s, "sizes")
			}
			if src, density = c.srcsetCandidate(GetAttr(s, "srcset"), sizes, ""); src != "" {
				return src, density
			}
		}
	}
	src = strings.TrimSpace(GetAttr(n, "src"))
	if s, d := c.srcsetCandidate(GetAttr(n, "srcset"), sizes, src); s != "" {
		return s, d
	}
	return src, 1
}

// imageTypeSupported returns whether images of the given MIME type (of a
// source element) can be shown, which they all can if it is not set.
func imageTypeSupported(typ string) bool {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "" {
		return true
	}
	switch typ {
	case "image/png", "image/jpeg", "image/jpg", "image/gif", "image/webp", "image/bmp", "image/svg+xml", "image/x-icon", "image/vnd.microsoft.icon":
		return true
	}
	return false
}

// srcsetCandidate returns the URL and density of the candidate of the
// given srcset that is best for the pixel ratio of the screen: the one of
// the least density that is at least that ratio, or else the one of the
// most density. The density of a width descriptor is its width over that
// of the image from the given sizes. The given src (of the img) is a
// candidate of 1x if the srcset has density descriptors and none of 1x, as
// in browsers (as the images of wikipedia.org, with a src and a srcset of 2x).
func (c *Context) srcsetCandidate(srcset, sizes, src string) (string, float32) {
	type candidate struct {
		url     string
		density float32
	}
	var cands []candidate
	var slot float32
	for _, item := range splitSrcset(srcset) {
		f := strings.Fields(item)
		if len(f) == 0 {
			continue
		}
		cd := candidate{url: f[0], density: 1}
		if len(f) > 1 {
			d := strings.ToLower(f[1])
			x, err := strconv.ParseFloat(d[:len(d)-1], 32)
			if err != nil || x <= 0 {
				continue
			}
			switch d[len(d)-1] {
			case 'x':
				cd.density = float32(x)
			case 'w':
				if slot == 0 {
					slot = c.imageSlotWidth(sizes)
				}
				cd.density = float32(x) / slot
			default:
				continue
			}
		}
		cands = append(cands, cd)
	}
	if len(cands) == 0 {
		return "", 0
	}
	if src != "" && slot == 0 && !slices.ContainsFunc(cands, func(cd candidate) bool { return cd.density == 1 }) {
		cands = append(cands, candidate{url: src, density: 1})
	}
	slices.SortStableFunc(cands, func(a, b candidate) int {
		switch {
		case a.density < b.density:
			return -1
		case a.density > b.density:
			return 1
		}
		return 0
	})
	ratio := devicePixelRatio() * max(c.Zoom, 1)
	for _, cd := range cands {
		if cd.density >= ratio-0.01 {
			return cd.url, cd.density
		}
	}
	last := cands[len(cands)-1]
	return last.url, last.density
}

// splitSrcset splits the given srcset into its candidates, at the commas
// that follow white space or a descriptor (not those in the URLs, as in
// the query of crop=3:2,... of an image CDN).
func splitSrcset(srcset string) []string {
	var items []string
	start := 0
	inURL := false // in the URL of the current candidate
	seen := false  // a URL has started
	for i, r := range srcset {
		switch {
		case r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f':
			if seen {
				inURL = false
			}
		case r == ',' && !inURL && seen:
			items = append(items, srcset[start:i])
			start, seen = i+1, false
		default:
			if !seen {
				seen, inURL = true, true
			}
		}
	}
	if seen {
		items = append(items, srcset[start:])
	}
	return items
}

// imageSlotWidth returns the width of an image in CSS px from the given
// sizes attribute: the length of its first entry whose media condition
// matches, or the width of the viewport.
func (c *Context) imageSlotWidth(sizes string) float32 {
	env := c.readEnv()
	for entry := range strings.SplitSeq(sizes, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		// the length is after the last white space outside of parentheses
		cond, length := "", entry
		depth := 0
		for i, r := range entry {
			switch {
			case r == '(':
				depth++
			case r == ')':
				depth--
			case depth == 0 && (r == ' ' || r == '\t' || r == '\n'):
				cond, length = strings.TrimSpace(entry[:i]), strings.TrimSpace(entry[i+1:])
			}
		}
		if cond != "" && !c.mediaMatches(cond) {
			continue
		}
		if length == "auto" {
			break
		}
		if px, ok := env.lengthPx(length); ok && px > 0 {
			return px
		}
	}
	return c.mediaWidth()
}

// devicePixelRatio returns the ratio of the pixels of the screen to CSS
// px, or 1 if there is no screen.
func devicePixelRatio() float32 {
	if system.TheApp != nil && system.TheApp.NScreens() > 0 {
		if r := system.TheApp.Screen(0).DevicePixelRatio; r > 0 {
			return r
		}
	}
	return 1
}
