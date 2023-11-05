// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log/slog"
	"os"

	"goki.dev/goki/config"
)

// AppJSTmpl is the template used in [MakeAppJS] to build the app.js file
var AppJSTmpl = template.Must(template.New("app.js").Parse(AppJS))

// AppJSData is the data passed to AppJSTmpl
type AppJSData struct {
	Env                     string
	LoadingLabel            string
	Wasm                    string
	WasmContentLengthHeader string
	WorkerJS                string
	AutoUpdateInterval      int64
}

// MakeAppJS exectues [AppJSTmpl] based on the given configuration information.
func MakeAppJS(c *config.Config) ([]byte, error) {
	if c.Web.Env == nil {
		c.Web.Env = make(map[string]string)
	}
	c.Web.Env["GOAPP_VERSION"] = c.Version
	c.Web.Env["GOAPP_STATIC_RESOURCES_URL"] = "/"
	c.Web.Env["GOAPP_ROOT_PREFIX"] = c.Build.Package

	for k, v := range c.Web.Env {
		if err := os.Setenv(k, v); err != nil {
			slog.Error("setting app env variable failed", "name", k, "value", "err", err)
		}
	}

	wenv, err := json.Marshal(c.Web.Env)
	if err != nil {
		return nil, err
	}

	d := AppJSData{
		Env:                     string(wenv),
		LoadingLabel:            c.Web.LoadingLabel,
		Wasm:                    "/app.wasm",
		WasmContentLengthHeader: c.Web.WasmContentLengthHeader,
		WorkerJS:                "/app-worker.js",
		AutoUpdateInterval:      c.Web.AutoUpdateInterval.Milliseconds(),
	}
	b := &bytes.Buffer{}
	err = AppJSTmpl.Execute(b, d)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// AppWorkerJSData is the data passed to [config.Config.Web.ServiceWorkerTemplate]
type AppWorkerJSData struct {
	Version          string
	ResourcesToCache string
}

// MakeWorkerJS executes [config.Config.Web.ServiceWorkerTemplate]. If it empty, it
// sets it to [DefaultAppWorkerJS].
func MakeAppWorkerJS(c *config.Config) ([]byte, error) {
	resources := []string{
		"/app.css",
		"/app.js",
		"/app.wasm",
		"/manifest.webmanifest",
		"/wasm_exec.js",
		"/",
	}

	if c.Web.ServiceWorkerTemplate == "" {
		c.Web.ServiceWorkerTemplate = DefaultAppWorkerJS
	}

	tmpl, err := template.New("app-worker.js").Parse(c.Web.ServiceWorkerTemplate)
	if err != nil {
		return nil, err
	}

	rstr, err := json.Marshal(resources)
	if err != nil {
		return nil, err
	}

	d := AppWorkerJSData{
		Version:          c.Version,
		ResourcesToCache: string(rstr),
	}

	b := &bytes.Buffer{}
	err = tmpl.Execute(b, d)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// ManifestJSONTmpl is the template used in [MakeManifestJSON] to build the mainfest.webmanifest file
var ManifestJSONTmpl = template.Must(template.New("manifest.webmanifest").Parse(ManifestJSON))

// ManifestJSONData is the data passed to [ManifestJSONTmpl]
type ManifestJSONData struct {
	ShortName       string
	Name            string
	Description     string
	DefaultIcon     string
	LargeIcon       string
	SVGIcon         string
	BackgroundColor string
	ThemeColor      string
	Scope           string
	StartURL        string
}

// MakeManifestJSON exectues [ManifestJSONTmpl] based on the given configuration information.
func MakeManifestJSON(c *config.Config) ([]byte, error) {
	d := ManifestJSONData{
		ShortName:   c.Name,
		Name:        c.Name,
		Description: c.Desc,
		// DefaultIcon:     h.Icon.Default,
		// LargeIcon:       h.Icon.Large,
		// SVGIcon:         h.Icon.SVG,
		BackgroundColor: c.Web.BackgroundColor,
		ThemeColor:      c.Web.ThemeColor,
		Scope:           "/",
		StartURL:        "/",
	}

	b := &bytes.Buffer{}
	err := ManifestJSONTmpl.Execute(b, d)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// IndexHTMLTmpl is the template used in [MakeIndexHTML] to build the index.html file
var IndexHTMLTmpl = template.Must(template.New("index.html").Parse(IndexHTML))

// IndexHTMLData is the data passed to [IndexHTMLTmpl]
type IndexHTMLData struct{}

// MakeIndexHTML exectues [IndexHTMLTmpl] based on the given configuration information.
func MakeIndexHTML(c *config.Config) ([]byte, error) {
	d := IndexHTMLData{}

	b := &bytes.Buffer{}
	err := IndexHTMLTmpl.Execute(b, d)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
