// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"

	"github.com/aymerick/douceur/css"
	"golang.org/x/net/html"
)

// The URLs of the resources that a page gets as it is read, for an app to
// get them in parallel before it is read (as the preload scanners of
// browsers do), as reading a page gets them one at a time: the first read
// of volkskrant.nl got 255 of them in 8s before it was shown.

// PrefetchURLs returns the absolute URLs of the resources that reading the
// given document with the context gets, as [Context.GetURL] gets them: its
// style sheets (and those that they import, and their web fonts, if they
// are inline; see [StyleSheetURLs] for those of the others), its scripts
// (if [Context.Scripting]), and its images (as chosen from their srcsets
// for the width of the context), in that order. It does not change the
// context.
func (c *Context) PrefetchURLs(doc *html.Node) []string {
	var sheets, scripts, images []string
	seen := map[string]bool{}
	add := func(list *[]string, raw, base string) {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "data:") || strings.HasPrefix(raw, "blob:") {
			return
		}
		u, err := parseRelativeURL(raw, base)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return
		}
		s := u.String()
		if !seen[s] {
			seen[s] = true
			*list = append(*list, s)
		}
	}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Namespace == "" {
			switch n.Data {
			case "link":
				rel := strings.ToLower(GetAttr(n, "rel"))
				switch {
				case rel == "stylesheet" && c.mediaMatches(GetAttr(n, "media")):
					add(&sheets, GetAttr(n, "href"), c.PageURL)
				case rel == "preload" || rel == "modulepreload":
					switch strings.ToLower(GetAttr(n, "as")) {
					case "style":
						add(&sheets, GetAttr(n, "href"), c.PageURL)
					case "script", "":
						if c.Scripting {
							add(&scripts, GetAttr(n, "href"), c.PageURL)
						}
					case "font":
						if c.WebFonts {
							add(&sheets, GetAttr(n, "href"), c.PageURL)
						}
					case "image":
						add(&images, GetAttr(n, "href"), c.PageURL)
					}
				}
			case "style":
				if c.mediaMatches(GetAttr(n, "media")) {
					for _, u := range StyleSheetURLs(nodeText(n), c.PageURL, c.WebFonts) {
						add(&sheets, u, c.PageURL)
					}
				}
			case "script":
				if c.Scripting && !HasAttr(n, "nomodule") {
					add(&scripts, GetAttr(n, "src"), c.PageURL)
				}
			case "img":
				src, _ := c.imageSource(n)
				add(&images, src, c.PageURL)
			}
		}
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			walk(k)
		}
	}
	walk(doc)
	return append(append(sheets, scripts...), images...)
}

// StyleSheetURLs returns the absolute URLs of the resources that the given
// style sheet, at the given URL (or in the page at it), gets as it is
// added to a page: the style sheets that it imports, and the web fonts of
// its @font-face rules (as [Context.addFontFace] chooses them), if
// webFonts and they have not been loaded yet. Its media and supports
// conditions are not checked.
func StyleSheetURLs(sheet, base string, webFonts bool) []string {
	var urls []string
	var walk func(rules []*css.Rule)
	walk = func(rules []*css.Rule) {
		for _, rule := range rules {
			if rule.Kind != css.AtRule {
				continue
			}
			switch strings.ToLower(rule.Name) {
			case "@import":
				if u, _ := importURL(rule.Prelude); u != "" {
					if pu, err := parseRelativeURL(u, base); err == nil {
						urls = append(urls, pu.String())
					}
				}
			case "@font-face":
				if webFonts {
					if u := fontFaceURL(rule.Declarations, base); u != "" {
						urls = append(urls, u)
					}
				}
			default:
				walk(rule.Rules) // (as @media and @supports)
			}
		}
	}
	walk(parseCSS(sheet).Rules)
	return urls
}

// fontFaceURL returns the URL of the web font that [Context.addFontFace]
// would get for the given declarations of an @font-face rule in a style
// sheet at the given URL, or "" if it would get none, or has gotten it.
func fontFaceURL(decls []*css.Declaration, base string) string {
	var family, src, urange string
	for _, d := range decls {
		switch strings.ToLower(strings.TrimSpace(d.Property)) {
		case "font-family":
			family = strings.Trim(strings.TrimSpace(d.Value), `"'`)
		case "src":
			src = d.Value
		case "unicode-range":
			urange = d.Value
		}
	}
	if family == "" || src == "" || (urange != "" && !unicodeRangeHas(urange, 'A')) {
		return ""
	}
	for _, s := range splitTopLevel(src, ',') {
		u, format := fontSource(s)
		if u == "" {
			continue
		}
		switch format {
		case "", "woff", "woff2", "truetype", "opentype", "woff-variations", "woff2-variations", "truetype-variations", "opentype-variations":
		default:
			continue
		}
		pu, err := parseRelativeURL(u, base)
		if err != nil {
			continue
		}
		key := strings.ToLower(family) + " " + pu.String()
		webFontURLs.Lock()
		_, tried := webFontURLs.loaded[key]
		webFontURLs.Unlock()
		if tried {
			return "" // (loaded once for all pages)
		}
		return pu.String()
	}
	return ""
}
