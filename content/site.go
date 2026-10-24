// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"fmt"
	"html"
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/content/bcontent"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/token"
	"cogentcore.org/core/tree"
)

// generateSite generates the JSON [bcontent.Site] data for the website
// of the content, which is used by the core build tool for both the
// pre-rendered preview shown while the wasm app is loading and for a
// freestanding website without the wasm app (htmlonly mode). Only the
// page content is converted to HTML; the navigation trees are generated
// from the returned data by the core build tool.
func generateSite(ct *Content) string {
	site := &bcontent.Site{}
	ct.UpdateTree() // need initial update first
	for _, pg := range ct.pages {
		if pg.Draft {
			continue
		}
		ct.Open(pg.URL)
		prp := &bcontent.PreRenderPage{
			Page:        *pg,
			HTML:        core.GenerateHTMLCore(ct.rightFrame),
			Description: pageDescription(ct),
			TOC:         pageTOC(ct.rightFrame),
		}
		site.Pages = append(site.Pages, prp)
	}
	for _, cat := range ct.categories {
		sc := &bcontent.Category{Name: cat}
		if cpg := ct.pageByName(cat); cpg != nil && !cpg.Draft {
			sc.HasPage = true
			sc.URL = cpg.URL
		}
		for _, pg := range ct.pagesByCategory[cat] {
			if pg.Draft || strings.EqualFold(cat, pg.Name) {
				continue
			}
			sc.Pages = append(sc.Pages, &bcontent.NavPage{Name: pg.Name, URL: pg.URL})
		}
		site.Categories = append(site.Categories, sc)
	}
	site.CSS = codeCSS()
	return string(errors.Log1(jsonx.WriteBytes(site)))
}

// pageDescription returns the description of the current page, which is the
// first non-emphasized paragraph (<em> typically indicates a note or caption,
// not an introduction).
func pageDescription(ct *Content) string {
	desc := ""
	ct.WalkDown(func(n tree.Node) bool {
		if desc != "" {
			return tree.Break
		}
		if tx, ok := n.(*core.Text); ok {
			if tx.Property("tag") == "p" && !strings.HasPrefix(tx.Text, "<em>") {
				desc = tx.Text
				return tree.Break
			}
		}
		return tree.Continue
	})
	return desc
}

// pageTOC returns the table of contents headings in the given page frame,
// consistent with [Content.makeTableOfContents].
func pageTOC(w *core.Frame) []*bcontent.TOCHeading {
	var toc []*bcontent.TOCHeading
	w.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		tx, ok := cw.(*core.Text)
		if !ok {
			return tree.Continue
		}
		tag, ok := tx.Property("tag").(string)
		if !ok || len(tag) != 2 || tag[0] != 'h' || tx.Property("page-title") != nil {
			return tree.Continue
		}
		level, err := strconv.Atoi(tag[1:])
		if err != nil {
			return tree.Continue
		}
		id, ok := tx.Property("id").(string)
		if !ok {
			id = tx.Name // consistent with [core.ToHTML]
		}
		toc = append(toc, &bcontent.TOCHeading{Level: level, Text: tx.Text, ID: id})
		return tree.Continue
	})
	return toc
}

// codeHTML returns syntax highlighted HTML for the given code in the given
// language, using the CSS classes from [codeCSS].
func codeHTML(code, lang string) string {
	var b strings.Builder
	b.WriteString(`<pre class="code"><code`)
	if lang != "" {
		fmt.Fprintf(&b, ` class="language-%s"`, strings.ToLower(lang))
	}
	b.WriteString(">")
	lx := lang
	if strings.EqualFold(lx, "goal") {
		lx = "go"
	}
	clex := highlighting.GetLexer(lx)
	lines := strings.Split(strings.TrimRight(code, "\n"), "\n")
	for i, ln := range lines {
		if i > 0 {
			b.WriteString("\n")
		}
		if clex == nil {
			b.WriteString(html.EscapeString(ln))
			continue
		}
		tags, err := highlighting.ChromaTagsLine(clex, ln)
		if err != nil {
			b.WriteString(html.EscapeString(ln))
			continue
		}
		b.Write(highlighting.MarkupLineHTML([]rune(ln), tags, nil, highlighting.EscapeHTML))
	}
	b.WriteString("</code></pre>")
	return b.String()
}

// codeCSS returns the CSS for the syntax highlighting classes used in
// [codeHTML], for both light and dark color schemes, based on
// [core.AppearanceSettings].
func codeCSS() string {
	isDark := matcolor.SchemeIsDark
	defer func() {
		colors.SetScheme(isDark)
		highlighting.UpdateFromTheme()
	}()
	var b strings.Builder
	for _, dark := range []bool{false, true} {
		colors.SetScheme(dark)
		highlighting.UpdateFromTheme()
		st := highlighting.AvailableStyle(core.AppearanceSettings.Highlighting)
		css := st.ToCSS()
		tags := make([]token.Tokens, 0, len(css))
		for tag := range css {
			tags = append(tags, tag)
		}
		slices.Sort(tags)
		indent := ""
		if dark {
			b.WriteString("@media (prefers-color-scheme: dark) {\n")
			indent = "  "
		}
		for _, tag := range tags {
			if css[tag] == "" {
				continue
			}
			fmt.Fprintf(&b, "%s.code .%s { %s; }\n", indent, tag.StyleName(), css[tag])
		}
		if dark {
			b.WriteString("}\n")
		}
	}
	return b.String()
}
