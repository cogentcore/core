// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
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
// Each element of a page is marked with a data-fid attribute, and the
// border box of each element in Chrome is compared with the box of the
// widget made for it. The results, with screenshots of both renderings
// side by side, are written to testdata/fidelity/out/report.html.

var (
	fidelityFlag = flag.Bool("fidelity", false, "run the fidelity comparison with Chrome")
	fidelityRun  = flag.String("fidelity-run", "", "regexp selecting the fidelity pages to compare")
	fidelityDir  = flag.String("fidelity-dir", "testdata/fidelity", "directory of the fidelity pages; a page.url file next to a page has the URL that its relative URLs are relative to")
)

const (
	// fidelityWidth is the width of the viewport, in CSS px (= dp = dots).
	fidelityWidth = 800

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
}

func TestFidelity(t *testing.T) {
	if !*fidelityFlag {
		t.Skip("run with -fidelity to compare with Chrome")
	}
	files, err := filepath.Glob(filepath.Join(*fidelityDir, "*.html"))
	if err != nil {
		t.Fatal(err)
	}
	var re *regexp.Regexp
	if *fidelityRun != "" {
		re = regexp.MustCompile(*fidelityRun)
	}
	outDir := filepath.Join("testdata", "fidelity", "out")
	os.MkdirAll(outDir, 0750)

	// The screen is tall enough for any page.
	system.TheApp.Screen(0).PixelSize = image.Pt(fidelityWidth, 6000)

	var results []*fidResult
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".html")
		if re != nil && !re.MatchString(name) {
			continue
		}
		res := fidelityPage(t, f, name, outDir)
		results = append(results, res)
		if res.Err != "" {
			t.Errorf("%s: %s", name, res.Err)
			continue
		}
		t.Logf("%-28s %3d/%-3d boxes match (%.0f%%), missing %d, height chrome %d core %d",
			name, res.OK, res.Total, 100*res.Score, res.Missing, res.ChromeH, res.CoreH)
	}
	writeFidelityReport(t, outDir, results)
}

// fidelityPage compares the rendering of one page.
func fidelityPage(t *testing.T, file, name, outDir string) *fidResult {
	res := &fidResult{Name: name}
	src, err := os.ReadFile(file)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	tagged, tags, err := tagFidelityHTML(src)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	absDir, _ := filepath.Abs(filepath.Dir(file))
	baseURL := ""
	if b, err := os.ReadFile(strings.TrimSuffix(file, ".html") + ".url"); err == nil {
		baseURL = strings.TrimSpace(string(b))
		tagged = strings.Replace(tagged, "<head>", `<head><base href="`+baseURL+`">`, 1)
	}

	chromeBoxes, chromeH, err := chromeRender(tagged, absDir, filepath.Join(outDir, name+".chrome.png"))
	if err != nil {
		res.Err = "chrome: " + err.Error()
		return res
	}
	res.ChromeH = chromeH

	coreBoxes, coreImg := coreRender(tagged, absDir, baseURL)
	if coreImg == nil {
		res.Err = "core: no image"
		return res
	}
	res.CoreH = coreImg.Bounds().Dy()
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
		el := &fidElem{Fid: f, Tag: tags[f], Chrome: chromeBoxes[f], Core: coreBoxes[f]}
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

// tagFidelityHTML adds a data-fid attribute to each element of the HTML,
// returning the HTML and the tag of each fid.
func tagFidelityHTML(src []byte) (string, map[int]string, error) {
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
	tags := map[int]string{}
	n := 0
	inBody := false
	var walk func(nd *html.Node)
	walk = func(nd *html.Node) {
		if nd.Type == html.ElementNode {
			if nd.Data == "body" {
				inBody = true
			}
			if inBody {
				n++
				tags[n] = nd.Data
				nd.Attr = append(nd.Attr, html.Attribute{Key: "data-fid", Val: strconv.Itoa(n)})
			}
		}
		for c := nd.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	var b bytes.Buffer
	if err := html.Render(&b, doc); err != nil {
		return "", nil, err
	}
	return b.String(), tags, nil
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

// chromeHarnessStyle makes Chrome use the same default font as Cogent Core.
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
	sb.WriteString("html { font-family: \"Noto Sans\"; }\n</style>\n")
	return sb.String()
}

// chromeHarnessCSP allows only the harness script to run in Chrome.
const chromeHarnessCSP = `<meta http-equiv="Content-Security-Policy" content="script-src 'nonce-fidelity'">`

const chromeHarnessScript = `<script nonce="fidelity">
document.fonts.ready.then(() => {
	const boxes = [];
	for (const e of document.querySelectorAll('[data-fid]')) {
		const r = e.getBoundingClientRect();
		boxes.push({Fid: +e.dataset.fid, Tag: e.tagName.toLowerCase(), Display: getComputedStyle(e).display,
			X: r.x, Y: r.y, W: r.width, H: r.height});
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
// elements and the height of the page, and saving a screenshot.
func chromeRender(tagged, dir, shot string) (map[int]*fidBox, int, error) {
	page := strings.Replace(tagged, "<head>", "<head>"+chromeHarnessCSP+chromeHarnessStyle(), 1)
	boxPage := strings.Replace(page, "</body>", chromeHarnessScript+"</body>", 1)
	tmp := filepath.Join(dir, ".fidelity-tmp.html")
	defer os.Remove(tmp)

	if err := os.WriteFile(tmp, []byte(boxPage), 0666); err != nil {
		return nil, 0, err
	}
	args := []string{"--headless", "--disable-gpu", "--hide-scrollbars", "--allow-file-access-from-files",
		"--force-device-scale-factor=1", "--virtual-time-budget=3000"}
	out, err := exec.Command(chromePath(), append(args, fmt.Sprintf("--window-size=%d,600", fidelityWidth),
		"--dump-dom", "file://"+tmp)...).Output()
	if err != nil {
		return nil, 0, err
	}
	m := fidelityJSON.FindSubmatch(out)
	if m == nil {
		return nil, 0, fmt.Errorf("no box output")
	}
	var data struct {
		H     int
		Boxes []*fidBox
	}
	if err := json.Unmarshal([]byte(html.UnescapeString(string(m[1]))), &data); err != nil {
		return nil, 0, err
	}
	boxes := map[int]*fidBox{}
	for _, b := range data.Boxes {
		boxes[b.Fid] = b
	}

	// screenshot without the script
	if err := os.WriteFile(tmp, []byte(page), 0666); err != nil {
		return nil, 0, err
	}
	h := max(data.H, 50)
	absShot, _ := filepath.Abs(shot)
	cmd := exec.Command(chromePath(), append(args, fmt.Sprintf("--window-size=%d,%d", fidelityWidth, h),
		"--screenshot="+absShot, "file://"+tmp)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, 0, fmt.Errorf("screenshot: %w: %s", err, out)
	}
	return boxes, h, nil
}

// coreRender renders the HTML with htmlcore, returning the boxes of the
// widgets of the elements and the rendered image.
func coreRender(tagged, dir, baseURL string) (map[int]*fidBox, *image.RGBA) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) {
		s.Padding.Zero()
		s.Gap.Zero()
		s.Min.X.Dp(fidelityWidth)
		s.Max.X.Dp(fidelityWidth)
		s.ScrollbarWidth.Zero() // as Chrome is run with hidden scrollbars
	})
	b.Scene.Styler(func(s *styles.Style) { // like the viewport of a browser
		s.Padding.Zero()
	})
	ctx := NewContext()
	ctx.BrowserStyle = true
	ctx.MediaWidth = fidelityWidth
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
		return nil, nil
	}
	boxes := map[int]*fidBox{}
	img := b.RenderImage(func() {
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
	})
	return boxes, img
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
<table><tr><th>page</th><th>match</th><th>missing</th><th>height chrome / core</th></tr>
{{range .Results}}<tr><td><a href="#{{.Name}}">{{.Name}}</a></td><td>{{.OK}}/{{.Total}} ({{pct .Score}})</td><td>{{.Missing}}</td><td>{{.ChromeH}} / {{.CoreH}}</td></tr>
{{end}}</table>
{{range .Results}}
<h2 id="{{.Name}}">{{.Name}} — {{pct .Score}}</h2>
{{if .Err}}<p>Error: {{.Err}}</p>{{else}}
<img src="{{.Image}}">
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
