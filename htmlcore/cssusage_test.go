// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

var cssUsageFlag = flag.Bool("css-usage", false, "report the CSS properties used by the pages of -fidelity-dir")

// supportedProperties are the CSS properties that [Context.setCSS] handles
// (and those of inline elements), for [TestCSSUsage].
var supportedProperties = map[string]bool{}

func init() {
	for _, p := range strings.Fields(`display margin margin-top margin-right margin-bottom margin-left
	padding padding-top padding-right padding-bottom padding-left border border-top border-right
	border-bottom border-left border-width border-style border-color border-top-width border-right-width
	border-bottom-width border-left-width border-top-style border-right-style border-bottom-style
	border-left-style border-top-color border-right-color border-bottom-color border-left-color
	border-top-left-radius border-top-right-radius border-bottom-right-radius border-bottom-left-radius
	border-radius background background-color color width height min-width min-height max-width
	max-height vertical-align border-collapse box-sizing font-size font-weight font-style font-family
	font line-height text-align white-space text-indent text-decoration text-decoration-line
	flex-direction flex-wrap flex-flow flex-grow flex gap grid-gap row-gap column-gap border-spacing
	justify-content justify-items justify-self align-items align-self align-content overflow overflow-x
	overflow-y grid-template-columns opacity visibility margin-inline margin-block padding-inline
	padding-block list-style list-style-type cursor`) {
		supportedProperties[p] = true
	}
	for p := range logicalProperties {
		supportedProperties[p] = true
	}
}

// TestCSSUsage reports the CSS properties (and the values of some) that
// apply to the elements of the pages in -fidelity-dir, with the number of
// elements and pages for each, marking those that are not supported, to
// find the features that matter most for real pages:
//
//	go test -run TestCSSUsage -css-usage -fidelity-dir testdata/fidelity/real -v
func TestCSSUsage(t *testing.T) {
	if !*cssUsageFlag {
		t.Skip("run with -css-usage")
	}
	files, _ := filepath.Glob(filepath.Join(*fidelityDir, "*.html"))
	type usage struct {
		elements int
		pages    map[string]bool
	}
	uses := map[string]*usage{}
	add := func(key, page string) {
		u := uses[key]
		if u == nil {
			u = &usage{pages: map[string]bool{}}
			uses[key] = u
		}
		u.elements++
		u.pages[page] = true
	}
	valued := map[string]bool{"display": true, "position": true, "float": true, "text-transform": true,
		"clear": true, "box-shadow": false, "transform": false}
	for _, f := range files {
		page := strings.TrimSuffix(filepath.Base(f), ".html")
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := html.ParseWithOptions(bytes.NewReader(src), html.ParseOptionEnableScripting(false))
		if err != nil {
			t.Fatal(err)
		}
		ctx := NewContext()
		ctx.BrowserStyle = true
		ctx.MediaWidth = float32(fidelityWidth())
		if b, err := os.ReadFile(strings.TrimSuffix(f, ".html") + ".url"); err == nil {
			ctx.PageURL = strings.TrimSpace(string(b))
		}
		ctx.Node = doc
		ctx.addStyleOrigin(browserStyles, originUserAgent)
		ctx.addStyleSheets(doc)
		var walk func(n *html.Node, hidden bool)
		walk = func(n *html.Node, hidden bool) {
			if n.Type == html.ElementNode {
				if ctx.displayOf(n) == "none" {
					hidden = true
				}
				if !hidden {
					seen := map[string]bool{}
					for _, d := range ctx.declarations(n) {
						if d.origin == originUserAgent || strings.HasPrefix(d.Property, "--") || seen[d.Property] {
							continue
						}
						seen[d.Property] = true
						key := d.Property
						if valued[key] {
							key += ": " + strings.ToLower(strings.Fields(d.Value + " x")[0])
						}
						add(key, page)
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c, hidden)
			}
		}
		walk(doc, false)
	}
	keys := make([]string, 0, len(uses))
	for k := range uses {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := uses[keys[i]], uses[keys[j]]
		if len(a.pages) != len(b.pages) {
			return len(a.pages) > len(b.pages)
		}
		return a.elements > b.elements
	})
	var sb strings.Builder
	for _, k := range keys {
		prop, _, _ := strings.Cut(k, ":")
		mark := "  "
		if !supportedProperties[prop] {
			mark = "- "
		}
		fmt.Fprintf(&sb, "%s%-40s %3d pages %6d elements\n", mark, k, len(uses[k].pages), uses[k].elements)
	}
	t.Log("\n" + sb.String())
}
