// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

// The snapshots in testdata/snapshots are real web pages saved with all of
// the resources that they need (style sheets and images), with their URLs
// rewritten to the local files, so that they render the same way without
// the network, for the regression tests ([TestPages]) and the comparison
// with Chrome ([TestFidelity] with -fidelity-dir testdata/snapshots).
// They are saved from the pages in testdata/fidelity/real-urls.txt with:
//
//	go test -run TestSnapshot -snapshot [-snapshot-run REGEXP]
//
// Scripts, iframes, and refreshes are removed (the pages work without
// JavaScript), and fonts are not saved (as Cogent Core uses its own),
// so nothing is loaded from the network in Chrome either.

var (
	snapshotFlag = flag.Bool("snapshot", false, "save the snapshots of the pages of real-urls.txt in testdata/snapshots")
	snapshotRun  = flag.String("snapshot-run", "", "regexp selecting the pages to save snapshots of")
)

// snapshotsDir is the directory of the snapshots, each in a directory
// of its own named for the page, with the page in index.html.
const snapshotsDir = "testdata/snapshots"

// snapshotMaxSize is the largest resource that is saved, in bytes.
const snapshotMaxSize = 4 << 20

// snapshotManifest is saved as manifest.json in the directory of a snapshot.
type snapshotManifest struct {
	URL       string
	Saved     time.Time
	Resources map[string]string // URL -> file, or the error of getting it
}

// snapshotter saves one page and its resources.
type snapshotter struct {
	dir      string
	manifest *snapshotManifest
	files    map[string]string // URL -> the local URL of its file
	local    map[string]bool   // the local URLs, relative to the page or a style sheet
}

func TestSnapshot(t *testing.T) {
	if !*snapshotFlag {
		t.Skip("run with -snapshot to save the snapshots of the real pages")
	}
	urls, err := os.ReadFile("testdata/fidelity/real-urls.txt")
	if err != nil {
		t.Fatal(err)
	}
	var re *regexp.Regexp
	if *snapshotRun != "" {
		re = regexp.MustCompile(*snapshotRun)
	}
	for _, line := range strings.Split(string(urls), "\n") {
		fs := strings.Fields(line)
		if len(fs) < 2 || strings.HasPrefix(fs[0], "#") || (re != nil && !re.MatchString(fs[0])) {
			continue
		}
		name, u := fs[0], fs[1]
		dir := filepath.Join(snapshotsDir, name)
		os.RemoveAll(dir)
		if err := os.MkdirAll(filepath.Join(dir, "files"), 0750); err != nil {
			t.Fatal(err)
		}
		sn := &snapshotter{dir: dir, files: map[string]string{}, local: map[string]bool{},
			manifest: &snapshotManifest{URL: u, Saved: time.Now().UTC(), Resources: map[string]string{}}}
		size, err := sn.savePage(u)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		mb, _ := json.MarshalIndent(sn.manifest, "", "\t")
		os.WriteFile(filepath.Join(dir, "manifest.json"), mb, 0666)
		t.Logf("%-16s %4d resources, %7d bytes", name, len(sn.manifest.Resources), size)
	}
}

// savePage saves the page at the given URL in index.html, with its
// resources, returning the total size of the files.
func (sn *snapshotter) savePage(u string) (int, error) {
	resp, err := getURL(u)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("status %s", resp.Status)
	}
	base := resp.Request.URL // after redirects
	// as htmlcore reads pages, without scripting: the content of noscript
	// elements is shown, as it is in Chrome in the fidelity test
	doc, err := html.ParseWithOptions(resp.Body, html.ParseOptionEnableScripting(false))
	if err != nil {
		return 0, err
	}
	unwrapNoscript(doc)
	var ebase *html.Node
	walkNodes(doc, func(n *html.Node) {
		if ebase == nil && n.Type == html.ElementNode && n.Data == "base" && GetAttr(n, "href") != "" {
			ebase = n
			if b, err := base.Parse(GetAttr(n, "href")); err == nil {
				base = b
			}
		}
	})
	if ebase != nil { // the URLs are all rewritten relative to the page
		ebase.Parent.RemoveChild(ebase)
	}
	sn.rewriteHTML(doc, base)
	var b bytes.Buffer
	if err := html.Render(&b, doc); err != nil {
		return 0, err
	}
	if err := os.WriteFile(filepath.Join(sn.dir, "index.html"), b.Bytes(), 0666); err != nil {
		return 0, err
	}
	size := b.Len()
	filepath.Walk(filepath.Join(sn.dir, "files"), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += int(info.Size())
		}
		return nil
	})
	return size, nil
}

// walkNodes calls the function for each node in the tree, in document
// order, before its children. The function can remove the node.
func walkNodes(n *html.Node, f func(n *html.Node)) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		f(c)
		if c.Parent != nil { // unless removed
			walkNodes(c, f)
		}
		c = next
	}
}

// rewriteHTML rewrites the URLs of the elements in the document, saving
// the resources that they need.
func (sn *snapshotter) rewriteHTML(doc *html.Node, base *url.URL) {
	walkNodes(doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		attr := func(name string, f func(v string) string) {
			for i, a := range n.Attr {
				if a.Key == name && a.Namespace == "" {
					n.Attr[i].Val = f(a.Val)
				}
			}
		}
		abs := func(v string) string { return absURL(base, v) }
		switch n.Data {
		case "script", "iframe", "frame", "embed", "object":
			n.Parent.RemoveChild(n)
			return
		case "meta":
			if strings.EqualFold(GetAttr(n, "http-equiv"), "refresh") {
				n.Parent.RemoveChild(n)
				return
			}
		case "link":
			rel := " " + strings.ToLower(GetAttr(n, "rel")) + " "
			switch {
			case strings.Contains(rel, " stylesheet "):
				attr("href", func(v string) string { return sn.saveCSSFile(base, v) })
			case strings.Contains(rel, "icon"):
				attr("href", func(v string) string { return sn.save(base, v) })
			case strings.Contains(rel, "preload") || strings.Contains(rel, "prefetch") ||
				strings.Contains(rel, "preconnect") || strings.Contains(rel, "manifest"):
				n.Parent.RemoveChild(n)
				return
			default:
				attr("href", abs)
			}
		case "style":
			if c := n.FirstChild; c != nil && c.Type == html.TextNode {
				c.Data = sn.rewriteCSS(base, c.Data, false)
			}
		case "img", "source", "input":
			if n.Data == "source" && n.Parent != nil && n.Parent.Data != "picture" {
				attr("src", func(v string) string { return noResource }) // of a video or audio
			} else {
				attr("src", func(v string) string { return sn.save(base, v) })
			}
			attr("srcset", func(v string) string { return sn.rewriteSrcset(base, v) })
		case "video", "audio", "track": // only the poster of a video is shown
			attr("src", func(v string) string { return noResource })
			attr("poster", func(v string) string { return sn.save(base, v) })
		case "image", "use", "feImage": // svg
			attr("href", func(v string) string { return sn.save(base, v) })
		case "a", "area":
			attr("href", abs)
		case "form":
			attr("action", abs)
		}
		if n.Data == "image" || n.Data == "use" {
			for i, a := range n.Attr {
				if a.Key == "href" && a.Namespace == "xlink" {
					n.Attr[i].Val = sn.save(base, a.Val)
				}
			}
		}
		attr("style", func(v string) string { return sn.rewriteCSS(base, v, false) })
		attr("background", func(v string) string { return sn.save(base, v) })
	})
}

// absURL returns the URL resolved relative to the base, unless it is
// only a fragment (a link within the page) or cannot be parsed.
func absURL(base *url.URL, v string) string {
	if strings.HasPrefix(v, "#") || v == "" {
		return v
	}
	u, err := base.Parse(strings.TrimSpace(v))
	if err != nil {
		return v
	}
	return u.String()
}

// noResource is the URL of a resource that is not saved, which loads
// nothing without the network.
const noResource = "data:,"

// save saves the resource at the URL (relative to the base), returning
// the local URL of its file, relative to the page.
func (sn *snapshotter) save(base *url.URL, v string) string {
	return sn.saveFile(base, v, nil)
}

// saveCSSFile saves the style sheet at the URL, with its URLs rewritten.
func (sn *snapshotter) saveCSSFile(base *url.URL, v string) string {
	return sn.saveFile(base, v, func(u *url.URL, b []byte) []byte {
		return []byte(sn.rewriteCSS(u, string(b), true))
	})
}

// saveFile saves the resource at the URL, rewriting its content with the
// function, if it is not nil, returning the local URL of its file.
func (sn *snapshotter) saveFile(base *url.URL, v string, rewrite func(u *url.URL, b []byte) []byte) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.HasPrefix(v, "data:") || strings.HasPrefix(v, "#") {
		return v
	}
	u, err := base.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return noResource
	}
	u.Fragment = ""
	key := u.String()
	if f, ok := sn.files[key]; ok {
		return f
	}
	sn.files[key] = noResource // while it is being saved, as for a style sheet that imports itself
	resp, err := getURL(key)
	if err != nil {
		sn.manifest.Resources[key] = "error: " + err.Error()
		return noResource
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, snapshotMaxSize+1))
	switch {
	case err != nil:
		sn.manifest.Resources[key] = "error: " + err.Error()
		return noResource
	case resp.StatusCode != 200:
		sn.manifest.Resources[key] = "error: " + resp.Status
		return noResource
	case len(b) > snapshotMaxSize:
		sn.manifest.Resources[key] = "error: too large"
		return noResource
	}
	if rewrite != nil {
		b = rewrite(resp.Request.URL, b)
	}
	sum := sha1.Sum([]byte(key))
	name := "files/" + hex.EncodeToString(sum[:6]) + resourceExt(u, resp.Header.Get("Content-Type"))
	if err := os.WriteFile(filepath.Join(sn.dir, name), b, 0666); err != nil {
		sn.manifest.Resources[key] = "error: " + err.Error()
		return noResource
	}
	sn.manifest.Resources[key] = name
	sn.files[key] = name
	return name
}

// resourceExt returns the file extension of a resource, from its URL or
// else its content type.
func resourceExt(u *url.URL, ctype string) string {
	ext := strings.ToLower(path.Ext(u.Path))
	if len(ext) > 1 && len(ext) <= 6 && ext != ".php" && ext != ".aspx" {
		return ext
	}
	mt, _, _ := mime.ParseMediaType(ctype)
	switch mt {
	case "text/css":
		return ".css"
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/svg+xml":
		return ".svg"
	case "image/webp":
		return ".webp"
	case "image/x-icon", "image/vnd.microsoft.icon":
		return ".ico"
	}
	if exts, _ := mime.ExtensionsByType(mt); len(exts) > 0 {
		sort.Strings(exts)
		return exts[0]
	}
	return ""
}

// rewriteSrcset rewrites the URLs of the candidates of a srcset.
func (sn *snapshotter) rewriteSrcset(base *url.URL, v string) string {
	var out []string
	for _, c := range strings.Split(v, ",") {
		fs := strings.Fields(c)
		if len(fs) == 0 {
			continue
		}
		fs[0] = sn.save(base, fs[0])
		out = append(out, strings.Join(fs, " "))
	}
	return strings.Join(out, ", ")
}

var (
	// cssImport matches an @import of a style sheet (by a string or url()).
	cssImport = regexp.MustCompile(`@import\s+(?:url\(\s*)?(["']?)([^"')\s;]+)(["']?)\s*\)?`)

	// cssURL matches a url() in CSS.
	cssURL = regexp.MustCompile(`url\(\s*(["']?)([^"')]*)(["']?)\s*\)`)

	// cssFontFace matches a @font-face rule.
	cssFontFace = regexp.MustCompile(`(?s)@font-face\s*\{[^}]*\}`)
)

// rewriteCSS rewrites the URLs in the CSS (relative to the base),
// saving the style sheets that it imports and the images that it uses,
// relative to the directory of the files if the CSS is in a file of its own
// (which is there), or else to the page. Fonts are not saved.
func (sn *snapshotter) rewriteCSS(base *url.URL, css string, inFile bool) string {
	rel := func(name string) string {
		if inFile {
			name = strings.TrimPrefix(name, "files/")
		}
		sn.local[name] = true
		return name
	}
	css = cssFontFace.ReplaceAllStringFunc(css, func(s string) string {
		return cssURL.ReplaceAllString(s, `url("`+noResource+`")`)
	})
	css = cssImport.ReplaceAllStringFunc(css, func(s string) string {
		m := cssImport.FindStringSubmatch(s)
		return `@import url("` + rel(sn.saveCSSFile(base, m[2])) + `")`
	})
	return cssURL.ReplaceAllStringFunc(css, func(s string) string {
		m := cssURL.FindStringSubmatch(s)
		v := m[2]
		if v == noResource || strings.HasPrefix(v, "data:") || strings.HasPrefix(v, "#") || sn.local[v] {
			return s
		}
		return `url("` + rel(sn.save(base, v)) + `")`
	})
}
