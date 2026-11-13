// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore/tohtml"
	"cogentcore.org/core/system"
)

var roundTripFlag = flag.Bool("roundtrip", false, "run the round trip comparison of the fidelity pages through tohtml with Chrome")

// roundTripPage is the page of the HTML that tohtml makes, with the styles
// of the pre-rendered pages of content sites (cmd/web/embed/content.css),
// without their fixed positioning of the site, so that the page is as tall
// as its content.
const roundTripPage = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><style>
{{CSS}}
body { overflow: visible; }
body>* { position: static; }
.cc-page { overflow: visible; }
</style></head><body><div class="cc-page">{{HTML}}</div></body></html>`

// rtResult is the round trip comparison of one page.
type rtResult struct {
	Name string

	// Core is the number of the boxes of the core rendering, and Same and
	// Orig are the numbers of the boxes of the round trip rendering that are
	// within the tolerance of the core and Chrome renderings.
	Core, Same, Orig int

	// Missing is the number of elements with a widget that are missing
	// in the round trip rendering.
	Missing int

	// Worst are the elements that differ most from the core rendering.
	Worst []string

	CoreH, RoundH int
	Err           string
}

// TestRoundTrip renders each fidelity page with htmlcore, converts the
// widgets back to HTML with tohtml, renders that in Chrome with the styles
// of the pre-rendered pages of content sites, and compares the boxes of the
// elements with those of the core rendering, as a check of how closely the
// HTML of tohtml looks like the widgets. Run it with
// go test -run TestRoundTrip -roundtrip [-fidelity-run RE] -v; it writes
// side by side images of the core and round trip renderings, and a
// summary, to testdata/fidelity/out/roundtrip.
func TestRoundTrip(t *testing.T) {
	if !*roundTripFlag {
		t.Skip("run with -roundtrip to compare with Chrome")
	}
	css, err := os.ReadFile(filepath.Join("..", "cmd", "web", "embed", "content.css"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(*fidelityDir, "*.html"))
	if err != nil {
		t.Fatal(err)
	}
	var re *regexp.Regexp
	if *fidelityRun != "" {
		re = regexp.MustCompile(*fidelityRun)
	}
	outDir := filepath.Join("testdata", "fidelity", "out", "roundtrip")
	os.MkdirAll(outDir, 0750)
	system.TheApp.Screen(0).PixelSize = image.Pt(fidelityWidth(), 6000)

	var results []*rtResult
	for _, f := range files {
		name := fidelityName(f)
		if re != nil && !re.MatchString(name) {
			continue
		}
		res := roundTripPage1(f, name, outDir, string(css))
		results = append(results, res)
		if res.Err != "" {
			t.Errorf("%s: %s", name, res.Err)
			continue
		}
		t.Logf("%-20s %3d/%-3d boxes as core, %3d as chrome, %d missing, height core %d round trip %d",
			name, res.Same, res.Core, res.Orig, res.Missing, res.CoreH, res.RoundH)
		for _, w := range res.Worst {
			t.Logf("    %s", w)
		}
	}
	writeRoundTripSummary(outDir, results)
}

// roundTripPage1 makes the round trip comparison of one page.
func roundTripPage1(file, name, outDir, css string) *rtResult {
	res := &rtResult{Name: name}
	src, err := os.ReadFile(file)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	tagged, fd, err := tagFidelityHTML(src)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	if _, err := os.Stat(strings.TrimSuffix(file, ".html") + ".url"); err == nil {
		res.Err = "skipped: a page on the web"
		return res
	}
	absDir, _ := filepath.Abs(filepath.Dir(file))

	// The original page in Chrome, for its viewport and boxes.
	chromeBoxes, _, viewport, err := chromeRender(tagged, absDir, filepath.Join(outDir, name+".chrome.png"), true)
	if err != nil {
		res.Err = "chrome: " + err.Error()
		return res
	}

	// The page in core, and the HTML of its widgets.
	var htm string
	coreRenderHook = func(b *core.Body) { htm = tohtml.Generate(b) }
	defer func() { coreRenderHook = nil }()
	coreBoxes, coreImg, coreH := coreRenderWidth(tagged, absDir, "", fidelityWidth(), viewport, func(u string) {})
	if coreImg == nil {
		res.Err = "core: no image"
		return res
	}
	res.CoreH = coreH
	savePNG(filepath.Join(outDir, name+".core.png"), coreImg)

	// The HTML of the widgets in Chrome.
	page := strings.Replace(strings.Replace(roundTripPage, "{{CSS}}", css, 1), "{{HTML}}", htm, 1)
	os.WriteFile(filepath.Join(outDir, name+".html"), []byte(page), 0666)
	rtBoxes, rtH, _, err := chromeRender(page, absDir, filepath.Join(outDir, name+".rt.png"), true)
	if err != nil {
		res.Err = "round trip chrome: " + err.Error()
		return res
	}
	res.RoundH = rtH
	savePNG(filepath.Join(outDir, name+".png"), sideBySide(coreImg, readPNG(filepath.Join(outDir, name+".rt.png"))))

	// Compare the boxes of the elements that have widgets.
	type diff struct {
		fid   int
		delta float32
	}
	var diffs []diff
	for fid, cb := range coreBoxes {
		if cb.W == 0 && cb.H == 0 {
			continue
		}
		res.Core++
		rb := rtBoxes[fid]
		if rb == nil || (rb.W == 0 && rb.H == 0) {
			res.Missing++
			continue
		}
		d := boxDelta(cb, rb)
		if d <= fidelityTolerance {
			res.Same++
		} else {
			diffs = append(diffs, diff{fid, d})
		}
		if ob := chromeBoxes[fid]; ob != nil && boxDelta(ob, rb) <= fidelityTolerance {
			res.Orig++
		}
	}
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].delta > diffs[j].delta })
	for _, d := range diffs[:min(len(diffs), 5)] {
		cb, rb := coreBoxes[d.fid], rtBoxes[d.fid]
		res.Worst = append(res.Worst, fmt.Sprintf("%d %s %s: core %.0f,%.0f %.0fx%.0f round trip %.0f,%.0f %.0fx%.0f",
			d.fid, fd.Tag[d.fid], fd.Desc[d.fid], cb.X, cb.Y, cb.W, cb.H, rb.X, rb.Y, rb.W, rb.H))
	}
	return res
}

// writeRoundTripSummary writes a summary of the round trip comparison.
func writeRoundTripSummary(outDir string, results []*rtResult) {
	var sb strings.Builder
	sb.WriteString("| page | as core | as chrome | missing | height core | height round trip |\n|---|---|---|---|---|---|\n")
	for _, r := range results {
		if r.Err != "" {
			fmt.Fprintf(&sb, "| %s | %s |\n", r.Name, r.Err)
			continue
		}
		fmt.Fprintf(&sb, "| %s | %d/%d | %d | %d | %d | %d |\n", r.Name, r.Same, r.Core, r.Orig, r.Missing, r.CoreH, r.RoundH)
	}
	os.WriteFile(filepath.Join(outDir, "summary.md"), []byte(sb.String()), 0666)
}
