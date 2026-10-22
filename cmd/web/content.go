// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/cmd/config"
	"cogentcore.org/core/content/bcontent"
	strip "github.com/grokify/html-strip-tags-go"
)

// buildHTMLOnly builds a freestanding website consisting only of static
// HTML pages generated from the content of the app, without the Go wasm app
// (see [config.Web.HTMLOnly]).
func buildHTMLOnly(c *config.Config) error {
	c.About = strip.StripTags(c.About)
	out, err := generateHTML()
	if err != nil {
		return err
	}
	site, err := readSite(out)
	if err != nil {
		return err
	}
	if site == nil {
		return errors.New("htmlonly: the app did not generate content site data; -htmlonly is only supported for apps that use the content package")
	}
	err = makeContentSite(c, site, false, "")
	if err != nil {
		return err
	}
	return makeIcons(c)
}

// generateHTML runs the app with the -generatehtml argument, which prints
// the pre-rendered HTML for the app, or the JSON [bcontent.Site] data
// for apps that use the content package.
func generateHTML() (string, error) {
	return exec.Output("go", "run", ".", "-generatehtml", "-nogui")
}

// readSite returns the [bcontent.Site] data in the given output of
// [generateHTML], or nil if the app does not use the content package.
func readSite(out string) (*bcontent.Site, error) {
	if !strings.HasPrefix(out, "{") {
		return nil, nil
	}
	site := &bcontent.Site{}
	err := jsonx.Read(site, strings.NewReader(out))
	if err != nil {
		return nil, err
	}
	return site, nil
}

// makeContentSite makes the website for the given content site data,
// with an index.html file for each page, which contains the pre-rendered
// page content along with the navigation trees and page search (using Vue.js)
// of the content package. If wasm is true, each page loads the Go wasm app,
// and the website is the preview shown while the app is loading. Otherwise,
// it is a freestanding website (see [config.Web.HTMLOnly]). If indexPage is
// non-empty, it is used for the root page instead (see [readIndexPage]).
func makeContentSite(c *config.Config, site *bcontent.Site, wasm bool, indexPage string) error {
	odir := c.Build.Output
	if c.Content == "" {
		c.Content = "content"
	}

	err := os.WriteFile(filepath.Join(odir, "content.css"), []byte(contentCSS+"\n/* Code highlighting */\n\n"+site.CSS), 0666)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "content.js"), []byte(contentJS), 0666)
	if err != nil {
		return err
	}
	nav, err := makeNavJS(c, site)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "nav.js"), nav, 0666)
	if err != nil {
		return err
	}

	// The content files that a js app loads as external embedded files are
	// shown from there by the pages too, so they are only downloaded once.
	var files map[string]string
	if wasm && IsJS(c) {
		files = externalContentFiles(c.Content, odir)
	}

	var root *bcontent.PreRenderPage
	for _, prp := range site.Pages {
		if prp.URL == "" {
			root = prp
		}
		pdir := filepath.Join(odir, filepath.FromSlash(prp.URL))
		err := os.MkdirAll(pdir, 0777)
		if err != nil {
			return err
		}
		var b []byte
		if prp.URL == "" && wasm && indexPage != "" {
			b, err = makeIndexHTML(c, "", &bcontent.PreRenderPage{Page: bcontent.Page{Name: c.Name}}, indexPage)
		} else {
			b, err = makeContentPage(c, site, prp, basePath(prp.URL), wasm, files)
		}
		if err != nil {
			return err
		}
		err = os.WriteFile(filepath.Join(pdir, "index.html"), b, 0666)
		if err != nil {
			return err
		}
	}

	// The 404 page loads the app for wasm, which can then handle the URL
	// appropriately, with the root page as the preview. Otherwise, it is
	// a page not found message. The base path is for pages one level below
	// the root (see [notFoundBasePath]).
	notFound := &bcontent.PreRenderPage{
		Page: bcontent.Page{Name: "Page not found", URL: "404"},
		HTML: `<div class="cc-page-frame"><h1 style="font-size:36px;font-weight:normal;line-height:1.2222222">Page not found</h1><p style="line-height:1.5">The requested page does not exist. Use the navigation or search to find another page.</p></div>`,
	}
	if wasm && root != nil {
		notFound = root
	}
	b, err := makeContentPage(c, site, notFound, notFoundBasePath(c), wasm, files)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "404.html"), b, 0666)
	if err != nil {
		return err
	}

	return copyContentFiles(c.Content, odir, files)
}

// basePath returns the relative path to the root of the website
// for the page with the given URL.
func basePath(url string) string {
	if url == "" {
		return ""
	}
	return strings.Repeat("../", strings.Count(strings.Trim(url, "/"), "/")+1)
}

// pageHref returns the link to the page with the given URL
// from a page with the given base path.
func pageHref(base, url string) string {
	if url == "" {
		if base == "" {
			return "./"
		}
		return base
	}
	return base + url + "/"
}

// navJSData is the site navigation data written to nav.js as window.ccSite,
// which is used by content.js for the categories tree and page search.
type navJSData struct {
	Name       string
	Pages      []*bcontent.NavPage
	Categories []*bcontent.Category
}

// makeNavJS returns the contents of nav.js for the given site.
func makeNavJS(c *config.Config, site *bcontent.Site) ([]byte, error) {
	d := &navJSData{Name: c.Name, Categories: site.Categories}
	for _, prp := range site.Pages {
		d.Pages = append(d.Pages, &bcontent.NavPage{Name: prp.Name, URL: prp.URL})
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return append(append([]byte("window.ccSite = "), b...), ";\n"...), nil
}

// contentTmpl is the template used in [makeContentPage].
var contentTmpl = template.Must(template.New("content.html").Parse(contentHTML))

// contentData is the data passed to [contentTmpl].
type contentData struct {
	indexHTMLData
	Wasm     bool
	Lang     string
	HomeURL  string
	NavHTML  string
	PageHTML string
	PageJSON string
}

// pageJSData is the page data in each page as window.ccPage,
// which is used by content.js for the table of contents.
type pageJSData struct {
	Name string
	URL  string
	TOC  []*bcontent.TOCHeading
}

// makeContentPage returns the HTML file for the given page of the site,
// with the given base path to the root of the site. If wasm is true,
// the page loads the Go wasm app (see [makeContentSite]). The given files
// map content files to other paths in the website (see [rebaseURLs]).
func makeContentPage(c *config.Config, site *bcontent.Site, prp *bcontent.PreRenderPage, base string, wasm bool, files map[string]string) ([]byte, error) {
	desc := c.About
	if prp.Description != "" && (prp.URL != "" || c.About == "") {
		desc = strings.TrimSpace(html.UnescapeString(strip.StripTags(prp.Description)))
	}
	title := prp.Name
	if prp.URL == "" { // root page just gets app name, as in content
		title = c.Name
	}
	pj, err := json.Marshal(&pageJSData{Name: prp.Name, URL: prp.URL, TOC: prp.TOC})
	if err != nil {
		return nil, err
	}
	d := &contentData{
		indexHTMLData: indexHTMLData{
			BasePath:               base,
			Author:                 c.Web.Author,
			Description:            html.EscapeString(desc),
			Keywords:               c.Web.Keywords,
			Title:                  html.EscapeString(title),
			SiteName:               html.EscapeString(c.Name),
			Image:                  c.Web.Image,
			Styles:                 c.Web.Styles,
			FontText:               html.EscapeString(c.Web.FontText),
			VanityURL:              c.Web.VanityURL,
			GithubVanityRepository: c.Web.GithubVanityRepository,
			JS:                     IsJS(c),
		},
		Wasm:     wasm,
		Lang:     c.Web.Lang,
		HomeURL:  pageHref(base, ""),
		NavHTML:  staticNavHTML(site, prp, base),
		PageHTML: rebaseURLs(prp.HTML, base, prp.URL, pageURLs(site), files),
		PageJSON: string(pj),
	}
	b := &bytes.Buffer{}
	err = contentTmpl.Execute(b, d)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// staticNavHTML returns the static navigation HTML for the given page,
// which is used as a fallback if JavaScript is not available, and which
// also makes all of the pages discoverable by search engines. It is
// replaced by the interactive Vue trees in content.js.
func staticNavHTML(site *bcontent.Site, prp *bcontent.PreRenderPage, base string) string {
	var b strings.Builder
	b.WriteString(`<div class="cc-static-nav">`)
	if len(prp.TOC) > 0 {
		b.WriteString(`<ul><li><b>Contents</b><ul>`)
		for _, h := range prp.TOC {
			fmt.Fprintf(&b, `<li style="margin-left:%dem"><a href="#%s">%s</a></li>`, h.Level-2, html.EscapeString(h.ID), h.Text)
		}
		b.WriteString(`</ul></li></ul>`)
	}
	if len(site.Categories) > 0 {
		fmt.Fprintf(&b, `<ul><li><a href="%s"><b>Categories</b></a><ul>`, pageHref(base, ""))
		for _, cat := range site.Categories {
			b.WriteString("<li>")
			if cat.HasPage {
				fmt.Fprintf(&b, `<a href="%s">%s</a>`, pageHref(base, cat.URL), html.EscapeString(cat.Name))
			} else {
				b.WriteString(html.EscapeString(cat.Name))
			}
			b.WriteString("<ul>")
			for _, pg := range cat.Pages {
				fmt.Fprintf(&b, `<li><a href="%s">%s</a></li>`, pageHref(base, pg.URL), html.EscapeString(pg.Name))
			}
			b.WriteString("</ul></li>")
		}
		b.WriteString(`</ul></li></ul>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

var (
	// htmlTagRe matches the start tags of HTML elements with URL attributes.
	htmlTagRe = regexp.MustCompile(`<(a|img|source|video|audio|iframe)\b[^>]*>`)

	// urlAttrRe matches a URL attribute in an HTML start tag.
	urlAttrRe = regexp.MustCompile(`\s(href|src)="([^"]*)"`)

	// urlSchemeRe matches URLs that start with a scheme (such as https:).
	urlSchemeRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
)

// pageURLs returns the set of the URLs of all of the pages in the site.
func pageURLs(site *bcontent.Site) map[string]bool {
	urls := map[string]bool{}
	for _, prp := range site.Pages {
		urls[prp.URL] = true
	}
	return urls
}

// rebaseURLs updates the relative URLs in the given page HTML, which are
// relative to the root of the content, to be relative to the page with the
// given URL at the given base path. Links to other pages are made to open
// in the same tab, and links to headings in the same page are reduced to
// just the heading fragment. Links to files (such as images or PDFs in the
// content directory) are left as links to the files themselves. A link is to
// a file if it is not one of the given page URLs and it has a file extension.
// The given files map the paths of content files to their paths in the
// website, if they are elsewhere (see [externalContentFiles]).
func rebaseURLs(h, base, pageURL string, pages map[string]bool, files map[string]string) string {
	return htmlTagRe.ReplaceAllStringFunc(h, func(tag string) string {
		internal := false // a link to a page, which opens in the same tab
		tag = urlAttrRe.ReplaceAllStringFunc(tag, func(attr string) string {
			sm := urlAttrRe.FindStringSubmatch(attr)
			name, u := sm[1], html.UnescapeString(sm[2])
			if u == "" && name == "href" { // the root page in content
				internal = true
				return fmt.Sprintf(` %s="%s"`, name, html.EscapeString(pageHref(base, "")))
			}
			if u == "" || u[0] == '#' || u[0] == '/' || urlSchemeRe.MatchString(u) {
				return attr
			}
			pg, frag, hasFrag := strings.Cut(u, "#")
			pg = strings.TrimSuffix(pg, "/")
			if name == "src" || (!pages[pg] && path.Ext(pg) != "") {
				if f, ok := files[u]; ok {
					u = f
				}
				return fmt.Sprintf(` %s="%s"`, name, html.EscapeString(base+u))
			}
			internal = true
			if hasFrag {
				frag = "#" + frag
				if pg == pageURL {
					return fmt.Sprintf(` %s="%s"`, name, html.EscapeString(frag))
				}
			}
			return fmt.Sprintf(` %s="%s"`, name, html.EscapeString(pageHref(base, pg)+frag))
		})
		if internal {
			tag = strings.Replace(tag, ` target="_blank"`, "", 1)
		}
		return tag
	})
}

// copyContentFiles copies all of the files in the given content directory
// other than the content pages themselves (such as images) to the
// given output directory, preserving their relative paths, except for
// the given files, which are elsewhere in the output directory.
func copyContentFiles(content, odir string, files map[string]string) error {
	if _, err := os.Stat(content); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	cfs := os.DirFS(content)
	return fs.WalkDir(cfs, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := path.Ext(p)
		if _, ok := files[p]; ok || ext == ".md" || ext == ".html" || strings.HasPrefix(path.Base(p), ".") {
			return nil
		}
		b, err := fs.ReadFile(cfs, p)
		if err != nil {
			return err
		}
		op := filepath.Join(odir, filepath.FromSlash(p))
		err = os.MkdirAll(filepath.Dir(op), 0777)
		if err != nil {
			return err
		}
		return os.WriteFile(op, b, 0666)
	})
}

// externalContentFiles returns the content files in the given content
// directory that the js app loads as external embedded files from the given
// output directory (see [copyExternalEmbeds]), mapped to their paths there.
// An external file is named by GoScript with the first 16 hexadecimal digits
// of the SHA-256 of its contents and its base name.
func externalContentFiles(content, odir string) map[string]string {
	files := map[string]string{}
	cfs := os.DirFS(content)
	fs.WalkDir(cfs, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		b, err := fs.ReadFile(cfs, p)
		if err != nil {
			return nil
		}
		sum := sha256.Sum256(b)
		name := hex.EncodeToString(sum[:])[:16] + "-" + path.Base(p)
		if _, err := os.Stat(filepath.Join(odir, jsExternalEmbedDir, name)); err == nil {
			files[p] = jsExternalEmbedDir + "/" + url.PathEscape(name)
		}
		return nil
	})
	return files
}
