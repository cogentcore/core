// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"golang.org/x/net/html"
)

// The analysis of the fidelity test finds where the layout of core starts
// to differ from that of Chrome, as the differences of the boxes compound:
// a paragraph that is one line higher moves all of the elements after it.
// An origin is an element whose box differs in a way that the boxes of the
// elements around it do not explain:
//
//   - width: its width differs, but that of its parent does not.
//   - height: its height differs, but those of its children do not
//     (or it has none: its text wraps differently, or an image is sized
//     differently).
//   - position: its position in its parent differs, but the position and
//     size of the element before it in the parent do not (or it is first).
//
// Separately, the computed styles of the elements are compared (font size
// and weight, margins, padding, and border widths), as a difference of
// those is of the cascade of the CSS, not of the layout. The margins of
// core can differ where margins collapse.
//
// The summary of each run is saved in testdata/fidelity/out/summary.json,
// and the report shows the changes from the summary of the previous run.

// fidOrigin is an element where a difference of the layout starts.
type fidOrigin struct {
	Fid   int
	Desc  string
	Kind  string // width, height, or position
	Delta float32
	Chrome, Core *fidBox
}

// fidStyleDiff is a difference of a computed style of an element.
type fidStyleDiff struct {
	Fid          int
	Desc         string
	Prop         string
	Chrome, Core float32
}

// fidPropCount is the number of elements with a difference of a property.
type fidPropCount struct {
	Prop  string
	Count int
}

// fidSummary is the summary of the comparison of a page, which is saved.
type fidSummary struct {
	Score          float32
	OK, Total      int
	Missing        int
	ChromeH, CoreH int
	Origins        int
	StyleDiffs     int
}

// elementDesc describes the element as a CSS selector: its tag, id,
// and classes (at most two).
func elementDesc(n *html.Node) string {
	d := n.Data
	if id := GetAttr(n, "id"); id != "" {
		d += "#" + id
	}
	cls := strings.Fields(GetAttr(n, "class"))
	if len(cls) > 2 {
		cls = cls[:2]
	}
	for _, c := range cls {
		d += "." + c
	}
	return d
}

// coreStyle returns the values of the properties of fidStyleProps for the
// style of a widget, in px (CSS px = dp, of which there are dp dots).
func coreStyle(s *styles.Style, dp float32) map[string]float32 {
	m := s.Margin.Dots()
	p := s.Padding.Dots()
	b := s.Border.Width.Dots()
	px := func(v float32) float32 { return math32.Round(v/dp*100) / 100 }
	return map[string]float32{
		"font-size":   px(s.Font.Size.Dots),
		"font-weight": s.Font.Weight.ToFloat32(),
		"margin-top":  px(m.Top), "margin-right": px(m.Right), "margin-bottom": px(m.Bottom), "margin-left": px(m.Left),
		"padding-top": px(p.Top), "padding-right": px(p.Right), "padding-bottom": px(p.Bottom), "padding-left": px(p.Left),
		"border-top-width": px(b.Top), "border-right-width": px(b.Right), "border-bottom-width": px(b.Bottom), "border-left-width": px(b.Left),
	}
}

// analyzeFidelity finds the origins of the differences of the layout and
// the differences of the styles of the elements of the result.
func analyzeFidelity(res *fidResult, fd *fidDoc) {
	els := map[int]*fidElem{}
	for _, el := range res.Elems {
		els[el.Fid] = el
	}
	// measured returns whether the element has a visible box in both
	measured := func(f int) bool {
		el := els[f]
		return el != nil && el.Chrome != nil && el.Core != nil && el.Chrome.W > 0 && el.Chrome.H > 0
	}
	// parent returns the nearest measured ancestor of the element, or 0
	parent := func(f int) int {
		for p := fd.Parent[f]; p > 0; p = fd.Parent[p] {
			if measured(p) {
				return p
			}
		}
		return 0
	}
	off := func(a, b float32) bool { return math32.Abs(a-b) > fidelityTolerance }
	children := map[int][]int{} // measured, in document order
	for _, el := range res.Elems {
		if measured(el.Fid) {
			p := parent(el.Fid)
			children[p] = append(children[p], el.Fid)
		}
	}
	for _, kids := range children {
		sort.Ints(kids)
	}
	before := map[int]int{} // the measured element before each in its parent
	for _, kids := range children {
		for i := 1; i < len(kids); i++ {
			before[kids[i]] = kids[i-1]
		}
	}
	relPos := func(f int) (cx, cy, kx, ky float32) {
		el := els[f]
		cx, cy, kx, ky = el.Chrome.X, el.Chrome.Y, el.Core.X, el.Core.Y
		if p := parent(f); p > 0 {
			pe := els[p]
			cx, cy, kx, ky = cx-pe.Chrome.X, cy-pe.Chrome.Y, kx-pe.Core.X, ky-pe.Core.Y
		}
		return
	}
	posOff := func(f int) bool {
		cx, cy, kx, ky := relPos(f)
		return off(cx, kx) || off(cy, ky)
	}
	sizeOff := func(f int) bool {
		el := els[f]
		return off(el.Chrome.W, el.Core.W) || off(el.Chrome.H, el.Core.H)
	}
	for _, el := range res.Elems {
		f := el.Fid
		if !measured(f) || el.Tag == "body" {
			continue
		}
		styleDiffs(res, el, fd)
		if el.Status == "ok" { // its box is right
			continue
		}
		add := func(kind string, delta float32) {
			res.Origins = append(res.Origins, &fidOrigin{Fid: f, Desc: fd.Desc[f], Kind: kind, Delta: delta, Chrome: el.Chrome, Core: el.Core})
		}
		if dw := el.Core.W - el.Chrome.W; off(el.Chrome.W, el.Core.W) {
			if p := parent(f); p == 0 || !off(els[p].Chrome.W, els[p].Core.W) {
				add("width", dw)
			}
		}
		if dh := el.Core.H - el.Chrome.H; off(el.Chrome.H, el.Core.H) {
			kidsOff := false
			for _, k := range children[f] {
				if off(els[k].Chrome.H, els[k].Core.H) {
					kidsOff = true
					break
				}
			}
			if !kidsOff {
				add("height", dh)
			}
		}
		if posOff(f) {
			if b, ok := before[f]; !ok || (!posOff(b) && !sizeOff(b)) {
				cx, cy, kx, ky := relPos(f)
				d := ky - cy
				if math32.Abs(kx-cx) > math32.Abs(d) {
					d = kx - cx
				}
				add("position", d)
			}
		}
	}
	counts := map[string]int{}
	for _, d := range res.StyleDiffs {
		counts[d.Prop]++
	}
	for p, c := range counts {
		res.PropCounts = append(res.PropCounts, fidPropCount{p, c})
	}
	sort.Slice(res.PropCounts, func(i, j int) bool {
		a, b := res.PropCounts[i], res.PropCounts[j]
		return a.Count > b.Count || (a.Count == b.Count && a.Prop < b.Prop)
	})
}

// styleDiffs adds the differences of the computed styles of the element.
func styleDiffs(res *fidResult, el *fidElem, fd *fidDoc) {
	if el.Chrome.Style == nil || el.Core.Style == nil {
		return
	}
	for _, p := range fidStyleProps {
		cv, kv := el.Chrome.Style[p], el.Core.Style[p]
		if math32.Abs(cv-kv) > 0.5 {
			res.StyleDiffs = append(res.StyleDiffs, &fidStyleDiff{Fid: el.Fid, Desc: fd.Desc[el.Fid], Prop: p, Chrome: cv, Core: kv})
		}
	}
}

// fidelitySummaries returns the summaries of the results.
func fidelitySummaries(results []*fidResult) map[string]*fidSummary {
	sums := map[string]*fidSummary{}
	for _, r := range results {
		if r.Err != "" {
			continue
		}
		sums[r.Name] = &fidSummary{Score: r.Score, OK: r.OK, Total: r.Total, Missing: r.Missing,
			ChromeH: r.ChromeH, CoreH: r.CoreH, Origins: len(r.Origins), StyleDiffs: len(r.StyleDiffs)}
	}
	return sums
}

// saveFidelitySummary saves the summaries of the results in summary.json
// in the directory, merged with those of the pages of the previous run
// that were not compared in this run, and returns the previous summaries.
func saveFidelitySummary(outDir string, results []*fidResult) map[string]*fidSummary {
	p := filepath.Join(outDir, "summary.json")
	prev := map[string]*fidSummary{}
	if b, err := os.ReadFile(p); err == nil {
		json.Unmarshal(b, &prev)
	}
	sums := fidelitySummaries(results)
	all := map[string]*fidSummary{}
	for n, s := range prev {
		all[n] = s
	}
	for n, s := range sums {
		all[n] = s
	}
	b, _ := json.MarshalIndent(all, "", "\t")
	os.WriteFile(p, b, 0666)
	return prev
}

// change returns the change of the score of the page from the previous
// summary, as text, or "" if there is none.
func (r *fidResult) change(prev map[string]*fidSummary) string {
	ps := prev[r.Name]
	if ps == nil || r.Err != "" {
		return ""
	}
	d := 100 * (r.Score - ps.Score)
	if math32.Abs(d) < 0.5 && r.Total == ps.Total {
		return "="
	}
	return fmt.Sprintf("%+.0f%% (was %d/%d)", d, ps.OK, ps.Total)
}

// logFidelity logs the main differences of the result.
func logFidelity(r *fidResult, logf func(format string, args ...any)) {
	n := min(len(r.Origins), 5)
	for _, o := range r.Origins[:n] {
		logf("    %-8s %+5.0f  %s (fid %d)", o.Kind, o.Delta, o.Desc, o.Fid)
	}
	if len(r.PropCounts) > 0 {
		var ps []string
		for _, pc := range r.PropCounts[:min(len(r.PropCounts), 6)] {
			ps = append(ps, fmt.Sprintf("%s %d", pc.Prop, pc.Count))
		}
		logf("    styles: %s", strings.Join(ps, ", "))
	}
}
