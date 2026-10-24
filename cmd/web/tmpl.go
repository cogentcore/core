// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bytes"
	"encoding/json"
	"html"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"cogentcore.org/core/base/elide"
	"cogentcore.org/core/cmd/config"
	"cogentcore.org/core/content/bcontent"
	strip "github.com/grokify/html-strip-tags-go"
)

// appJSTmpl is the template used in [makeAppJS] to build the app.js file
var appJSTmpl = template.Must(template.New("app.js").Parse(appJS))

// appJSData is the data passed to [appJSTmpl]
type appJSData struct {
	Env                     string
	WasmContentLengthHeader string
	AutoUpdateInterval      int64

	// JS is whether the app is compiled to JavaScript with GoScript
	// (the js platform) instead of to WebAssembly.
	JS bool
}

// makeAppJS exectues [appJSTmpl] based on the given configuration information.
func makeAppJS(c *config.Config) ([]byte, error) {
	if c.Web.Env == nil {
		c.Web.Env = make(map[string]string)
	}
	c.Web.Env["GOAPP_STATIC_RESOURCES_URL"] = "/"
	c.Web.Env["GOAPP_ROOT_PREFIX"] = "."

	for k, v := range c.Web.Env {
		if err := os.Setenv(k, v); err != nil {
			slog.Error("setting app env variable failed", "name", k, "value", v, "err", err)
		}
	}

	wenv, err := json.Marshal(c.Web.Env)
	if err != nil {
		return nil, err
	}

	d := appJSData{
		Env:                     string(wenv),
		WasmContentLengthHeader: c.Web.WasmContentLengthHeader,
		AutoUpdateInterval:      c.Web.AutoUpdateInterval.Milliseconds(),
		JS:                      IsJS(c),
	}
	b := &bytes.Buffer{}
	err = appJSTmpl.Execute(b, d)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// appWorkerJSData is the data passed to [config.Config.Web.ServiceWorkerTemplate]
type appWorkerJSData struct {
	Version                  string
	ResourcesToCache         string
	ExternalResourcesToCache string
	CacheOrigins             string
}

// makeAppWorkerJS executes [config.Config.Web.ServiceWorkerTemplate]. If it empty, it
// sets it to [appWorkerJS]. The service worker caches the files of the app
// for offline use, including those of the website of a content app if
// content is true (see [makeContentSite]).
func makeAppWorkerJS(c *config.Config, content bool) ([]byte, error) {
	resources := []string{
		"./",
		"index.html",
		"app.css",
		"app.js",
		"manifest.webmanifest",
		"icons/32.png",
		"icons/192.png",
		"icons/svg.svg",
	}
	if IsJS(c) {
		resources = append(resources, "app.gs.js")
		resources = append(resources, jsChunkFiles(c.Build.Output)...)
		resources = append(resources, externalEmbedFiles(c.Build.Output)...)
	} else {
		resources = append(resources, "app.wasm", "wasm_exec.js")
	}
	external := externalResources(indexHTML)
	if content {
		resources = append(resources, "content.css", "content.js", "nav.js")
		external = append(external, externalResources(contentHTML)...)
	}
	external = append(external, c.Web.Styles...)
	slices.Sort(external)
	external = slices.Compact(external)
	origins := []string{}
	for _, r := range external {
		u, err := url.Parse(r)
		if err != nil || u.Host == "" {
			continue
		}
		origins = append(origins, u.Scheme+"://"+u.Host)
	}
	// Google Fonts stylesheets load their fonts from this other origin.
	origins = append(origins, "https://fonts.gstatic.com")
	slices.Sort(origins)
	origins = slices.Compact(origins)

	tmpl, err := template.New("app-worker.js").Parse(appWorkerJS)
	if err != nil {
		return nil, err
	}

	d := appWorkerJSData{Version: c.Version}
	for _, v := range []struct {
		dst *string
		src []string
	}{{&d.ResourcesToCache, resources}, {&d.ExternalResourcesToCache, external}, {&d.CacheOrigins, origins}} {
		b, err := json.Marshal(v.src)
		if err != nil {
			return nil, err
		}
		*v.dst = string(b)
	}

	b := &bytes.Buffer{}
	err = tmpl.Execute(b, d)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

var (
	// externalTagRe matches the link and script start tags in an HTML template.
	externalTagRe = regexp.MustCompile(`<(?:link|script)\b[^>]*>`)

	// externalURLRe matches a URL from another origin in a link or script tag,
	// with the URL as its first submatch.
	externalURLRe = regexp.MustCompile(`\s(?:href|src)="(https://[^"{]+)"`)
)

// externalResources returns the URLs of the stylesheets and scripts from
// other origins (CDNs) that the given HTML template loads.
func externalResources(tmpl string) []string {
	var urls []string
	for _, tag := range externalTagRe.FindAllString(tmpl, -1) {
		if strings.HasPrefix(tag, "<link") && !strings.Contains(tag, `rel="stylesheet"`) {
			continue
		}
		m := externalURLRe.FindStringSubmatch(tag)
		if m != nil {
			urls = append(urls, m[1])
		}
	}
	return urls
}

// manifestJSONTmpl is the template used in [makeManifestJSON] to build the mainfest.webmanifest file
var manifestJSONTmpl = template.Must(template.New("manifest.webmanifest").Parse(manifestJSON))

// manifestJSONData is the data passed to [manifestJSONTmpl]
type manifestJSONData struct {
	ShortName   string
	Name        string
	Description string
}

// makeManifestJSON exectues [manifestJSONTmpl] based on the given configuration information.
func makeManifestJSON(c *config.Config) ([]byte, error) {
	d := manifestJSONData{
		ShortName:   elide.AppName(c.Name),
		Name:        c.Name,
		Description: c.About,
	}

	b := &bytes.Buffer{}
	err := manifestJSONTmpl.Execute(b, d)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// indexHTMLTmpl is the template used in [makeIndexHTML] to build the index.html file
var indexHTMLTmpl = template.Must(template.New("index.html").Parse(indexHTML))

// indexHTMLData is the data passed to [indexHTMLTmpl]
type indexHTMLData struct {
	BasePath               string
	Author                 string
	Description            string
	Keywords               []string
	Title                  string
	SiteName               string
	Image                  string
	Styles                 []string
	FontText               string
	VanityURL              string
	GithubVanityRepository string
	PreRenderHTML          string
	IndexPage              string

	// JS is whether the app is compiled to JavaScript with GoScript
	// (the js platform), in which case wasm_exec.js is not needed.
	JS bool
}

// makeIndexHTML exectues [indexHTMLTmpl] based on the given configuration information,
// base path for app resources (used in [makePages]), optional title (used in [makePages],
// defaults to [config.Config.Name] otherwise), optional page-specific description (used
// in [makePages], defaults to [config.Config.About]), and pre-render HTML representation
// of app content. If indexPage is non-empty, it is the HTML for the custom index page
// of the app (see [readIndexPage]), which is used instead of the pre-render HTML.
func makeIndexHTML(c *config.Config, basePath string, prp *bcontent.PreRenderPage, indexPage string) ([]byte, error) {
	if prp.Description == "" {
		prp.Description = c.About
	} else {
		// c.About is already stripped earlier, so only necessary
		// for page-specific description here.
		prp.Description = strip.StripTags(prp.Description)
	}
	d := indexHTMLData{
		BasePath:               basePath,
		Author:                 c.Web.Author,
		Description:            prp.Description,
		Keywords:               c.Web.Keywords,
		Title:                  prp.Name,
		SiteName:               c.Name,
		Image:                  c.Web.Image,
		Styles:                 c.Web.Styles,
		FontText:               html.EscapeString(c.Web.FontText),
		VanityURL:              c.Web.VanityURL,
		GithubVanityRepository: c.Web.GithubVanityRepository,
		PreRenderHTML:          prp.HTML,
		IndexPage:              indexPage,
		JS:                     IsJS(c),
	}

	b := &bytes.Buffer{}
	err := indexHTMLTmpl.Execute(b, d)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
