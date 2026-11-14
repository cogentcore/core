// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
	"golang.org/x/net/html"
)

// The fidelity test compares the rendering of the HTML pages in
// testdata/fidelity with that of Chrome, which it runs headless. It is run
// with the -fidelity flag (and -fidelity-run to select pages by a regexp):
//
//	go test -run TestFidelity -fidelity
//
// The snapshots of real pages (see [TestSnapshot]) are compared with:
//
//	go test -run TestFidelity -fidelity -fidelity-dir testdata/snapshots -fidelity-width 1024
//
// Each element of a page is marked with a data-fid attribute, and the
// border box of each element in Chrome is compared with the box of the
// widget made for it. Chrome uses the fonts of Cogent Core (see
// chromeFontScript), and local pages are rendered without the network in
// both. The results, with screenshots of both renderings side by side, the
// origins of the differences of the layouts, and the differences of the
// computed styles (see analyzeFidelity), are written to
// testdata/fidelity/out/report.html, with the analysis of each page in
// name.json there, and the changes from the previous run in the report.

var (
	fidelityFlag      = flag.Bool("fidelity", false, "run the fidelity comparison with Chrome")
	fidelityRun       = flag.String("fidelity-run", "", "regexp selecting the fidelity pages to compare")
	fidelityWidthFlag = flag.Int("fidelity-width", 800, "width of the viewport of the fidelity comparison, in CSS px")
	fidelityDir       = flag.String("fidelity-dir", "testdata/fidelity", "directory of the fidelity pages; a page.url file next to a page has the URL that its relative URLs are relative to")
)

const (
	// fidelityTolerance is the largest difference in each edge of a box,
	// in px, for the box to be counted as matching.
	fidelityTolerance = 6
)

// chromePath returns the path of the Chrome executable.
func chromePath() string {
	if p := os.Getenv("CHROME"); p != "" {
		return p
	}
	return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
}

// fidBox is the box of an element, in one of the renderings.
type fidBox struct {
	Fid     int
	Tag     string
	Display string // chrome only
	Widget  string // core only
	X, Y    float32
	W, H    float32

	// Style has the computed values of the CSS properties in
	// fidStyleProps, in px (font-weight is a number).
	Style map[string]float32
}

func (b *fidBox) rect() image.Rectangle {
	return image.Rect(int(b.X), int(b.Y), int(b.X+b.W), int(b.Y+b.H))
}

// fidElem is the comparison of one element.
type fidElem struct {
	Fid    int
	Tag    string
	Chrome *fidBox
	Core   *fidBox
	Status string // ok, off, missing, inline, extra
	Delta  float32
}

// fidResult is the comparison of one page.
type fidResult struct {
	Name       string
	Elems      []*fidElem
	OK, Total  int
	Score      float32
	Image      string
	ChromeH    int
	CoreH      int
	Err        string
	Missing    int
	InlineOnly int

	// Origins are the elements where the differences of the layout start
	// (see analyzeFidelity), in document order.
	Origins []*fidOrigin

	// StyleDiffs are the differences of the computed styles.
	StyleDiffs []*fidStyleDiff

	// PropCounts are the numbers of StyleDiffs of each property.
	PropCounts []fidPropCount

	// Change is the change of the score from the previous run.
	Change string
}

func TestFidelity(t *testing.T) {
	if !*fidelityFlag {
		t.Skip("run with -fidelity to compare with Chrome")
	}
	files, err := filepath.Glob(filepath.Join(*fidelityDir, "*.html"))
	if err != nil {
		t.Fatal(err)
	}
	// and the pages in directories of their own, as the snapshots
	dirs, _ := filepath.Glob(filepath.Join(*fidelityDir, "*", "index.html"))
	files = append(files, dirs...)
	var re *regexp.Regexp
	if *fidelityRun != "" {
		re = regexp.MustCompile(*fidelityRun)
	}
	outDir := filepath.Join("testdata", "fidelity", "out")
	os.MkdirAll(outDir, 0750)

	// The screen is tall enough for any page.
	system.TheApp.Screen(0).PixelSize = image.Pt(fidelityWidth(), 6000)

	var results []*fidResult
	for _, f := range files {
		name := fidelityName(f)
		if re != nil && !re.MatchString(name) {
			continue
		}
		res := fidelityPage(t, f, name, outDir)
		results = append(results, res)
		if res.Err != "" {
			t.Errorf("%s: %s", name, res.Err)
			continue
		}
	}
	prev := saveFidelitySummary(outDir, results)
	for _, res := range results {
		if res.Err != "" {
			continue
		}
		res.Change = res.change(prev)
		t.Logf("%-20s %4d/%-4d boxes match (%.0f%%) %s, missing %d, height chrome %d core %d, %d origins, %d style diffs",
			res.Name, res.OK, res.Total, 100*res.Score, res.Change, res.Missing, res.ChromeH, res.CoreH, len(res.Origins), len(res.StyleDiffs))
		logFidelity(res, t.Logf)
	}
	writeFidelityReport(t, outDir, results)
}

// fidelityName returns the name of the fidelity page in the given file:
// its name, or the name of its directory for an index.html.
func fidelityName(file string) string {
	if filepath.Base(file) == "index.html" {
		return filepath.Base(filepath.Dir(file))
	}
	return strings.TrimSuffix(filepath.Base(file), ".html")
}

// fidelityPage compares the rendering of one page.
func fidelityPage(t *testing.T, file, name, outDir string) *fidResult {
	res := &fidResult{Name: name}
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
	absDir, _ := filepath.Abs(filepath.Dir(file))
	baseURL := ""
	if b, err := os.ReadFile(strings.TrimSuffix(file, ".html") + ".url"); err == nil {
		baseURL = strings.TrimSpace(string(b))
		tagged = insertInHead(tagged, `<base href="`+baseURL+`">`)
	}

	// a local page (without a base URL on the web) is rendered without the
	// network, so that it renders the same at any time
	offline := baseURL == ""
	chromeBoxes, chromeH, viewport, err := chromeRender(tagged, absDir, filepath.Join(outDir, name+".chrome.png"), offline)
	if err != nil {
		res.Err = "chrome: " + err.Error()
		return res
	}
	res.ChromeH = chromeH

	var remote func(u string)
	if offline {
		remote = func(u string) { t.Logf("%s: a resource on the web is not read: %s", name, u) }
	}
	coreBoxes, coreImg, coreH := coreRenderWidth(tagged, absDir, baseURL, fidelityWidth(), viewport, remote)
	if coreImg == nil {
		res.Err = "core: no image"
		return res
	}
	res.CoreH = coreH
	savePNG(filepath.Join(outDir, name+".core.png"), coreImg)

	// side by side image
	chromeImg := readPNG(filepath.Join(outDir, name+".chrome.png"))
	res.Image = name + ".png"
	savePNG(filepath.Join(outDir, res.Image), sideBySide(chromeImg, coreImg))

	// compare boxes
	fids := map[int]bool{}
	for f := range chromeBoxes {
		fids[f] = true
	}
	for f := range coreBoxes {
		fids[f] = true
	}
	var order []int
	for f := range fids {
		order = append(order, f)
	}
	sort.Ints(order)
	for _, f := range order {
		el := &fidElem{Fid: f, Tag: fd.Tag[f], Chrome: chromeBoxes[f], Core: coreBoxes[f]}
		switch {
		case el.Chrome == nil:
			el.Status = "extra"
		case el.Chrome.W == 0 && el.Chrome.H == 0:
			el.Status = "hidden"
			if el.Core != nil && el.Core.W > 0 && el.Core.H > 0 {
				el.Status = "off"
				res.Total++
			}
		case el.Core == nil:
			if strings.HasPrefix(el.Chrome.Display, "table-") && el.Chrome.Display != "table-cell" && el.Chrome.Display != "table-caption" {
				el.Status = "inline" // rows have no widgets: their cells are in the grid of the table
				res.InlineOnly++
			} else if strings.HasPrefix(el.Chrome.Display, "inline") && el.Chrome.Display != "inline-block" {
				el.Status = "inline"
				res.InlineOnly++
			} else {
				el.Status = "missing"
				res.Missing++
				res.Total++
			}
		default:
			res.Total++
			el.Delta = boxDelta(el.Chrome, el.Core)
			if el.Tag == "body" { // its height and position depend on margins collapsing through it
				el.Delta = max(math32.Abs(el.Chrome.X-el.Core.X), math32.Abs(el.Chrome.W-el.Core.W))
			}
			if el.Delta <= fidelityTolerance {
				el.Status = "ok"
				res.OK++
			} else {
				el.Status = "off"
			}
		}
		res.Elems = append(res.Elems, el)
	}
	if res.Total > 0 {
		res.Score = float32(res.OK) / float32(res.Total)
	}
	analyzeFidelity(res, fd)
	if b, err := json.MarshalIndent(map[string]any{"Origins": res.Origins, "StyleDiffs": res.StyleDiffs}, "", "\t"); err == nil {
		os.WriteFile(filepath.Join(outDir, name+".json"), b, 0666)
	}
	return res
}

// boxDelta returns the largest difference of an edge of the boxes.
func boxDelta(a, b *fidBox) float32 {
	d := math32.Abs(a.X - b.X)
	d = max(d, math32.Abs(a.Y-b.Y))
	d = max(d, math32.Abs((a.X+a.W)-(b.X+b.W)))
	d = max(d, math32.Abs((a.Y+a.H)-(b.Y+b.H)))
	return d
}

// chromeTimeout is the longest that a run of Chrome can take, as for a
// screenshot of a very long page, after which it is stopped.
const chromeTimeout = 2 * time.Minute

// fidelityWidth returns the width of the viewport, in CSS px (= dp = dots),
// from the -fidelity-width flag.
func fidelityWidth() int { return *fidelityWidthFlag }

// fidDoc has the elements of a fidelity page, by their fid.
type fidDoc struct {
	// Tag is the tag of each element.
	Tag map[int]string

	// Desc describes each element: its tag, id, and classes, as in CSS.
	Desc map[int]string

	// Parent is the fid of the parent element of each element (0 for body).
	Parent map[int]int
}

// tagFidelityHTML adds a data-fid attribute to each element of the body of
// the HTML (from 1 for the body), returning the HTML and its elements.
func tagFidelityHTML(src []byte) (string, *fidDoc, error) {
	// The pages are compared as browsers without JavaScript render them,
	// which is how htmlcore renders them: the content of noscript elements
	// is in the page (it is unwrapped, as Chrome is run with JavaScript only
	// for the harness, so it would not show it), and the page scripts are
	// blocked by a Content-Security-Policy (see chromeRender).
	doc, err := html.ParseWithOptions(bytes.NewReader(src), html.ParseOptionEnableScripting(false))
	if err != nil {
		return "", nil, err
	}
	unwrapNoscript(doc)
	fd := &fidDoc{Tag: map[int]string{}, Desc: map[int]string{}, Parent: map[int]int{}}
	n := 0
	inBody := false
	var walk func(nd *html.Node, parent int)
	walk = func(nd *html.Node, parent int) {
		if nd.Type == html.ElementNode {
			if nd.Data == "body" {
				inBody = true
			}
			if inBody {
				n++
				fd.Tag[n] = nd.Data
				fd.Desc[n] = elementDesc(nd)
				fd.Parent[n] = parent
				parent = n
				nd.Attr = append(nd.Attr, html.Attribute{Key: "data-fid", Val: strconv.Itoa(n)})
			}
		}
		for c := nd.FirstChild; c != nil; c = c.NextSibling {
			walk(c, parent)
		}
	}
	walk(doc, 0)
	var b bytes.Buffer
	if err := html.Render(&b, doc); err != nil {
		return "", nil, err
	}
	return b.String(), fd, nil
}

// headTag matches the start tag of the head element, with any attributes.
var headTag = regexp.MustCompile(`(?i)<head(\s[^>]*)?>`)

// insertInHead returns the page with the given HTML at the start of its head.
func insertInHead(page, s string) string {
	loc := headTag.FindStringIndex(page)
	if loc == nil {
		return page
	}
	return page[:loc[1]] + s + page[loc[1]:]
}

// unwrapNoscript replaces the noscript elements under n with their children.
func unwrapNoscript(n *html.Node) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if c.Type == html.ElementNode && c.Data == "noscript" {
			for gc := c.FirstChild; gc != nil; {
				gnext := gc.NextSibling
				c.RemoveChild(gc)
				n.InsertBefore(gc, c)
				unwrapNoscript(gc)
				gc = gnext
			}
			n.RemoveChild(c)
		} else {
			unwrapNoscript(c)
		}
		c = next
	}
}

// chromeHarnessStyle has the fonts of Cogent Core, Noto Sans (the default)
// and Roboto Mono, for Chrome (see chromeFontScript).
func chromeHarnessStyle() string {
	fonts, _ := filepath.Abs("../text/fonts/noto")
	var sb strings.Builder
	sb.WriteString("<style>\n")
	for _, f := range []struct {
		file, weight, style string
	}{
		{"Regular", "400", "normal"}, {"Italic", "400", "italic"},
		{"Medium", "500", "normal"}, {"MediumItalic", "500", "italic"},
		{"Bold", "700", "normal"}, {"BoldItalic", "700", "italic"},
	} {
		fmt.Fprintf(&sb, "@font-face { font-family: \"Noto Sans\"; src: url(\"file://%s/NotoSans-%s.ttf\"); font-weight: %s; font-style: %s; }\n",
			fonts, f.file, f.weight, f.style)
	}
	mono, _ := filepath.Abs("../text/fonts/robotomono")
	for _, f := range []struct {
		file, weight, style string
	}{
		{"Regular", "400", "normal"}, {"Italic", "400", "italic"},
		{"Medium", "500", "normal"}, {"MediumItalic", "500", "italic"},
		{"Bold", "700", "normal"}, {"BoldItalic", "700", "italic"},
	} {
		fmt.Fprintf(&sb, "@font-face { font-family: \"Roboto Mono\"; src: url(\"file://%s/RobotoMono-%s.ttf\"); font-weight: %s; font-style: %s; }\n",
			mono, f.file, f.weight, f.style)
	}
	sb.WriteString("html { font-family: \"Noto Sans\"; }\n</style>\n")
	return sb.String()
}

// chromeHarnessCSP allows only the harness script to run in Chrome.
const chromeHarnessCSP = `<meta http-equiv="Content-Security-Policy" content="script-src 'nonce-fidelity'">`

// chromeFontScript makes Chrome use the fonts that Cogent Core uses for the
// font families of the elements (see fontFamily): Roboto Mono for monospace
// fonts, and Noto Sans for all others, so that the differences of the
// layouts are not those of the fonts. It defines fidelityFonts, a promise
// of when the fonts are set and loaded.
const chromeFontScript = `<script nonce="fidelity">
const fidelityFonts = new Promise(resolve => addEventListener('DOMContentLoaded', async () => {
	const mono = /^(monospace|courier|courier new|consolas|menlo|monaco|roboto mono|ui-monospace|sfmono-regular)$/;
	const known = /^(serif|times|times new roman|georgia|garamond|ui-serif|sans-serif|arial|helvetica|helvetica neue|verdana|tahoma|system-ui|-apple-system|segoe ui|roboto|noto sans|ui-sans-serif|cursive|fantasy)$/;
	const coreFont = ff => {
		for (let f of ff.split(',')) {
			f = f.trim().replace(/^["']|["']$/g, '').toLowerCase();
			if (mono.test(f)) return '"Roboto Mono"';
			if (known.test(f)) return '"Noto Sans"';
		}
		return '"Noto Sans"';
	};
	const els = [document.documentElement, ...document.documentElement.querySelectorAll('*')];
	const fonts = els.map(e => coreFont(getComputedStyle(e).fontFamily)); // before any is set
	els.forEach((e, i) => e.style && e.style.setProperty('font-family', fonts[i], 'important'));
	await document.fonts.ready;
	resolve();
}));
</script>
`

// fidStyleProps are the CSS properties whose computed values are compared.
var fidStyleProps = []string{"font-size", "font-weight",
	"margin-top", "margin-right", "margin-bottom", "margin-left",
	"padding-top", "padding-right", "padding-bottom", "padding-left",
	"border-top-width", "border-right-width", "border-bottom-width", "border-left-width"}

// chromeHarnessScript gets the boxes and styles of the elements, and the
// height of the page, which it puts in a pre element at the end of the body.
const chromeHarnessScript = `<script nonce="fidelity">
fidelityFonts.then(() => {
	const props = PROPS;
	const boxes = [];
	for (const e of document.querySelectorAll('[data-fid]')) {
		const r = e.getBoundingClientRect();
		const cs = getComputedStyle(e);
		const st = {};
		for (const p of props) st[p] = parseFloat(cs.getPropertyValue(p)) || 0;
		boxes.push({Fid: +e.dataset.fid, Tag: e.tagName.toLowerCase(), Display: cs.display,
			X: r.x, Y: r.y, W: r.width, H: r.height, Style: st});
	}
	const h = Math.ceil(Math.max(document.documentElement.getBoundingClientRect().height, document.documentElement.scrollHeight > innerHeight ? document.documentElement.scrollHeight : 0));
	const pre = document.createElement('pre');
	pre.id = '__fidelity';
	pre.textContent = JSON.stringify({H: h, Boxes: boxes});
	document.body.appendChild(pre);
});
</script>
`

var fidelityJSON = regexp.MustCompile(`(?s)<pre id="__fidelity">(.*?)</pre>`)

// chromeRender renders the HTML in Chrome, returning the boxes of the
// elements, the height of the page, and the height of the viewport, and
// saving a screenshot. The viewport is as high as the page in a viewport
// of 600px, as for the screenshot, so that vh lengths are the same in the
// boxes as in the screenshot (and in the rendering of core, which uses it).
func chromeRender(tagged, dir, shot string, offline bool) (map[int]*fidBox, int, int, error) {
	props, _ := json.Marshal(fidStyleProps)
	page := insertInHead(tagged, chromeHarnessCSP+chromeHarnessStyle()+chromeFontScript)
	boxPage := strings.Replace(page, "</body>", strings.Replace(chromeHarnessScript, "PROPS", string(props), 1)+"</body>", 1)
	tmp := filepath.Join(dir, ".fidelity-tmp.html")
	defer os.Remove(tmp)

	args := []string{"--headless", "--disable-gpu", "--hide-scrollbars", "--allow-file-access-from-files",
		"--force-device-scale-factor=1", "--virtual-time-budget=3000",
		"--user-agent=" + UserAgent} // as htmlcore; some sites block headless Chrome
	if offline { // no host can be resolved
		args = append(args, "--host-resolver-rules=MAP * ~NOTFOUND")
	}
	run := func(page string, height int, out ...string) ([]byte, error) {
		if err := os.WriteFile(tmp, []byte(page), 0666); err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), chromeTimeout)
		defer cancel()
		a := append(append(args, fmt.Sprintf("--window-size=%d,%d", fidelityWidth(), height)), out...)
		cmd := exec.CommandContext(ctx, chromePath(), append(a, "file://"+tmp)...)
		if len(out) > 0 && out[0] == "--dump-dom" {
			return cmd.Output()
		}
		b, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, b)
		}
		return b, nil
	}
	type chromeData struct {
		H     int
		Boxes []*fidBox
	}
	boxesAt := func(height int) (*chromeData, error) {
		out, err := run(boxPage, height, "--dump-dom")
		if err != nil {
			return nil, err
		}
		m := fidelityJSON.FindSubmatch(out)
		if m == nil {
			return nil, fmt.Errorf("no box output")
		}
		data := &chromeData{}
		if err := json.Unmarshal([]byte(html.UnescapeString(string(m[1]))), data); err != nil {
			return nil, err
		}
		return data, nil
	}
	data, err := boxesAt(600)
	if err != nil {
		return nil, 0, 0, err
	}
	viewport := max(data.H, 50)
	if viewport != 600 {
		if data, err = boxesAt(viewport); err != nil {
			return nil, 0, 0, err
		}
	}
	boxes := map[int]*fidBox{}
	for _, b := range data.Boxes {
		boxes[b.Fid] = b
	}
	absShot, _ := filepath.Abs(shot)
	if _, err := run(page, viewport, "--screenshot="+absShot); err != nil { // without the box script
		return nil, 0, 0, fmt.Errorf("screenshot: %w", err)
	}
	return boxes, max(data.H, 50), viewport, nil
}

// coreRenderHook, if set, is called with the body of each page that
// [coreRenderWidth] renders, after its layout (see TestRoundTrip).
var coreRenderHook func(b *core.Body)

// coreRenderWidth renders the HTML with htmlcore in a viewport of the given
// width and height (for @media rules and vh lengths), returning the boxes of
// the widgets of the elements with a data-fid attribute, the rendered image,
// and the height of the page. If remote is not nil, the resources of the
// page are only read from local files, and it is called with the URL of
// each resource on the web that the page requests instead.
func coreRenderWidth(tagged, dir, baseURL string, width, height int, remote func(u string)) (map[int]*fidBox, *image.RGBA, int) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) {
		s.Padding.Zero()
		s.Gap.Zero()
		s.Min.X.Dp(float32(width))
		s.Max.X.Dp(float32(width))
		s.ScrollbarWidth.Zero() // as Chrome is run with hidden scrollbars
	})
	b.Scene.Styler(func(s *styles.Style) { // like the viewport of a browser
		s.Padding.Zero()
	})
	ctx := NewContext()
	ctx.MediaWidth = float32(width)
	ctx.MediaHeight = float32(height)
	ctx.PageURL = "file://" + dir + "/page.html"
	if baseURL != "" {
		ctx.PageURL = baseURL
	}
	ctx.GetURL = func(u string) (*http.Response, error) {
		pu, err := url.Parse(u)
		if err != nil {
			return nil, err
		}
		if pu.Scheme == "http" || pu.Scheme == "https" {
			if remote != nil {
				remote(u)
				return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader(""))}, nil
			}
			return getURL(u) // as the default GetURL does
		}
		p := pu.Path
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		f, err := os.Open(p)
		if err != nil {
			return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: f, Request: &http.Request{URL: pu}}, nil
	}
	if err := ReadHTMLString(ctx, b, tagged); err != nil {
		return nil, nil, 0
	}
	boxes := map[int]*fidBox{}
	pageHeight := 0 // in dp, as laid out (the window can be higher)
	img := b.RenderImage(func() {
		dp := b.Styles.UnitContext.Dots(units.UnitDp)
		bottom := float32(0) // of the content, as the body grows to fill the window
		b.ForWidgetChildren(func(i int, cw core.Widget, cwb *core.WidgetBase) bool {
			bottom = max(bottom, cwb.Geom.Pos.Total.Y+cwb.Geom.Size.Actual.Total.Y)
			return tree.Continue
		})
		pageHeight = int(math32.Ceil(bottom / dp))
		b.WalkDown(func(n tree.Node) bool {
			w, ok := n.(core.Widget)
			if !ok {
				return tree.Continue
			}
			wb := w.AsWidget()
			fs, ok := wb.Property("data-fid").(string)
			if !ok {
				return tree.Continue
			}
			fid, err := strconv.Atoi(fs)
			if err != nil {
				return tree.Continue
			}
			m := wb.Styles.Margin.Dots()
			pos := wb.Geom.Pos.Total
			sz := wb.Geom.Size.Actual.Total
			dp := wb.Styles.UnitContext.Dots(units.UnitDp) // boxes are in dp = CSS px
			bx := &fidBox{Fid: fid, Widget: fmt.Sprintf("%T", w),
				X: (pos.X + m.Left) / dp, Y: (pos.Y + m.Top) / dp,
				W: (sz.X - m.Left - m.Right) / dp, H: (sz.Y - m.Top - m.Bottom) / dp}
			if tag, ok := wb.Property("tag").(string); ok {
				bx.Tag = tag
			}
			// the box of a list item is that of its content (see
			// handleBrowserListItem), and its margins are of the item
			sw := w
			if pw, ok := wb.Parent.(core.Widget); ok && wb.Name == "content" && pw.AsWidget().Property("tag") == "li" {
				sw = pw
			}
			bx.Style = coreStyle(&sw.AsWidget().Styles, dp)
			if bs := ctx.blocks[sw]; bs != nil && bs.markers { // in the padding of the list
				uc := &sw.AsWidget().Styles.UnitContext
				bx.Style["padding-left"] += math32.Round(uc.ToDots(markerWidth.Value, markerWidth.Unit)/dp*100) / 100
			}
			if bs := ctx.blocks[sw]; bs != nil { // before the margins collapse
				bx.Style["margin-top"] = math32.Round(bs.marginTop/dp*100) / 100
				bx.Style["margin-bottom"] = math32.Round(bs.marginBottom/dp*100) / 100
			}
			if wb.Styles.Display == styles.DisplayNone || wb.StateIs(states.Invisible) {
				bx.W, bx.H = 0, 0
			}
			if old := boxes[fid]; old != nil { // union of the widgets of an element
				r := old.rect().Union(bx.rect())
				old.X, old.Y, old.W, old.H = float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
			} else {
				boxes[fid] = bx
			}
			return tree.Continue
		})
		if coreRenderHook != nil {
			coreRenderHook(b)
		}
	})
	if img != nil && pageHeight > 0 && pageHeight < img.Bounds().Dy() {
		img = img.SubImage(image.Rect(0, 0, img.Bounds().Dx(), pageHeight)).(*image.RGBA)
	}
	return boxes, img, pageHeight
}

func readPNG(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	return img
}

func savePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	png.Encode(f, img)
}

// sideBySide returns the images side by side, with a gap between them.
func sideBySide(a, b image.Image) image.Image {
	const gap = 12
	ab, bb := a.Bounds(), b.Bounds()
	w := ab.Dx() + gap + bb.Dx()
	h := max(ab.Dy(), bb.Dy())
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{128, 128, 128, 255}), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 0, ab.Dx(), ab.Dy()), a, ab.Min, draw.Src)
	draw.Draw(img, image.Rect(ab.Dx()+gap, 0, w, bb.Dy()), b, bb.Min, draw.Src)
	return img
}

var fidelityReport = template.Must(template.New("report").Funcs(template.FuncMap{
	"pct": func(f float32) string { return fmt.Sprintf("%.0f%%", 100*f) },
	"box": func(b *fidBox) string {
		if b == nil {
			return "—"
		}
		return fmt.Sprintf("%.0f,%.0f %.0f×%.0f", b.X, b.Y, b.W, b.H)
	},
	"round": func(f float32) string { return strconv.Itoa(int(math.Round(float64(f)))) },
}).Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>htmlcore fidelity</title>
<style>
body { font-family: system-ui, sans-serif; margin: 16px; color: #222; background: #fff; }
table { border-collapse: collapse; font-size: 13px; }
td, th { border: 1px solid #ccc; padding: 2px 6px; text-align: left; }
.ok { background: #e6f4e6; } .off { background: #fde8d8; } .missing { background: #f8d0d0; }
.inline, .hidden, .extra { color: #888; }
img { max-width: 100%; border: 1px solid #aaa; }
details { margin: 8px 0 24px; }
h2 { margin-top: 32px; }
</style></head><body>
<h1>htmlcore fidelity vs Chrome</h1>
<p>Each page: Chrome on the left, Cogent Core on the right. Boxes are border boxes in px; an element matches when every edge is within {{.Tol}}px.</p>
<p>Origins are the elements where differences of the layout start: a width that differs from Chrome while that of the parent does not, a height that differs while those of the children do not, or a position in the parent that differs while the element before it matches. Style differences are of the computed values of the CSS (the cascade), separate from the layout.</p>
<table><tr><th>page</th><th>match</th><th>change</th><th>missing</th><th>height chrome / core</th><th>origins</th><th>style diffs</th></tr>
{{range .Results}}<tr><td><a href="#{{.Name}}">{{.Name}}</a></td><td>{{.OK}}/{{.Total}} ({{pct .Score}})</td><td>{{.Change}}</td><td>{{.Missing}}</td><td>{{.ChromeH}} / {{.CoreH}}</td><td>{{len .Origins}}</td><td>{{len .StyleDiffs}}</td></tr>
{{end}}</table>
{{range .Results}}
<h2 id="{{.Name}}">{{.Name}} — {{pct .Score}}</h2>
{{if .Err}}<p>Error: {{.Err}}</p>{{else}}
<img src="{{.Image}}">
{{if .Origins}}<details open><summary>origins ({{len .Origins}})</summary>
<table><tr><th>fid</th><th>element</th><th>kind</th><th>Δ</th><th>chrome</th><th>core</th></tr>
{{range .Origins}}<tr><td>{{.Fid}}</td><td>{{.Desc}}</td><td>{{.Kind}}</td><td>{{round .Delta}}</td><td>{{box .Chrome}}</td><td>{{box .Core}}</td></tr>
{{end}}</table></details>{{end}}
{{if .StyleDiffs}}<details><summary>style differences: {{range .PropCounts}}{{.Prop}} {{.Count}}; {{end}}</summary>
<table><tr><th>fid</th><th>element</th><th>property</th><th>chrome</th><th>core</th></tr>
{{range .StyleDiffs}}<tr><td>{{.Fid}}</td><td>{{.Desc}}</td><td>{{.Prop}}</td><td>{{.Chrome}}</td><td>{{.Core}}</td></tr>
{{end}}</table></details>{{end}}
<details><summary>boxes</summary>
<table><tr><th>fid</th><th>tag</th><th>status</th><th>chrome</th><th>core</th><th>Δ</th><th>display / widget</th></tr>
{{range .Elems}}<tr class="{{.Status}}"><td>{{.Fid}}</td><td>{{.Tag}}</td><td>{{.Status}}</td><td>{{box .Chrome}}</td><td>{{box .Core}}</td><td>{{round .Delta}}</td><td>{{if .Chrome}}{{.Chrome.Display}}{{end}} / {{if .Core}}{{.Core.Widget}}{{end}}</td></tr>
{{end}}</table></details>{{end}}
{{end}}
</body></html>
`))

func writeFidelityReport(t *testing.T, outDir string, results []*fidResult) {
	var b bytes.Buffer
	err := fidelityReport.Execute(&b, map[string]any{"Results": results, "Tol": fidelityTolerance})
	if err != nil {
		t.Error(err)
		return
	}
	p := filepath.Join(outDir, "report.html")
	os.WriteFile(p, b.Bytes(), 0666)
	t.Logf("report: %s", p)
}
