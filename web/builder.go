// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"goki.dev/goki/config"
)

type builder struct {
	config.Config

	etag            string
	cachedResources *memoryCache
}

// ProxyResource is a proxy descriptor that maps a given resource to an URL
// path.
type ProxyResource struct {
	// The URL path from where a static resource is accessible.
	Path string

	// The path of the static resource that is proxied. It must start with
	// "/web/".
	ResourcePath string
}

func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Errorf("converting value to json string failed: %w", err))
	}
	return string(b)
}

const (
	defaultThemeColor = "#2d2c2c"
	packagePath       = "/"
	staticPath        = "/web"
	appWASMPath       = "/app.wasm"
)

func (h *builder) init() {
	h.initVersion()
	// h.initStaticResources()
	h.initImage()
	// h.initLibraries()
	// h.initLinks()
	// h.initScripts()
	h.initServiceWorker()
	// h.initCacheableResources()
	// h.initIcon()
	h.initPWA()
	// h.initPageContent()
	h.initPWAResources()
	h.initProxyResources()
}

func (h *builder) initVersion() {
	if h.Version == "" {
		t := time.Now().UTC().String()
		h.Version = fmt.Sprintf(`%x`, sha1.Sum([]byte(t)))
	}
	h.etag = `"` + h.Version + `"`
}

func (h *builder) initImage() {
	if h.Web.Image != "" {
		h.Web.Image = h.resolveStaticPath(h.Web.Image)
	}
}

func (h *builder) initServiceWorker() {
	if h.Web.ServiceWorkerTemplate == "" {
		h.Web.ServiceWorkerTemplate = DefaultAppWorkerJS
	}
}

func (h *builder) initPWA() {
	if h.Web.BackgroundColor == "" {
		h.Web.BackgroundColor = defaultThemeColor
	}
	if h.Web.ThemeColor == "" {
		h.Web.ThemeColor = defaultThemeColor
	}

	if h.Web.Lang == "" {
		h.Web.Lang = "en"
	}

	if h.Web.LoadingLabel == "" {
		h.Web.LoadingLabel = "{progress}%"
	}
}

func (h *builder) initPWAResources() {
	h.cachedResources = newMemoryCache(6)

	h.cachedResources.Set(cacheItem{
		Path:        "/wasm_exec.js",
		ContentType: "application/javascript",
		Body:        []byte(wasmExecJS()),
	})

	h.cachedResources.Set(cacheItem{
		Path:        "/app.js",
		ContentType: "application/javascript",
		Body:        h.makeAppJS(),
	})

	h.cachedResources.Set(cacheItem{
		Path:        "/app-worker.js",
		ContentType: "application/javascript",
		Body:        h.makeAppWorkerJS(),
	})

	h.cachedResources.Set(cacheItem{
		Path:        "/manifest.webmanifest",
		ContentType: "application/manifest+json",
		Body:        h.makeManifestJSON(),
	})

	h.cachedResources.Set(cacheItem{
		Path:        "/app.css",
		ContentType: "text/css",
		Body:        []byte(appCSS),
	})
	h.cachedResources.Set(cacheItem{
		Path:        "/",
		ContentType: "text/html",
		Body:        h.makeIndexHTML(),
	})
}

func (h *builder) makeAppJS() []byte {
	if h.Web.Env == nil {
		h.Web.Env = make(map[string]string)
	}
	h.Web.Env["GOAPP_VERSION"] = h.Version
	h.Web.Env["GOAPP_STATIC_RESOURCES_URL"] = staticPath
	h.Web.Env["GOAPP_ROOT_PREFIX"] = h.Build.Package

	for k, v := range h.Web.Env {
		if err := os.Setenv(k, v); err != nil {
			slog.Error("setting app env variable failed", "name", k, "value", "err", err)
		}
	}

	var b bytes.Buffer
	if err := template.
		Must(template.New("app.js").Parse(appJS)).
		Execute(&b, struct {
			Env                     string
			LoadingLabel            string
			Wasm                    string
			WasmContentLengthHeader string
			WorkerJS                string
			AutoUpdateInterval      int64
		}{
			Env:                     jsonString(h.Web.Env),
			LoadingLabel:            h.Web.LoadingLabel,
			Wasm:                    appWASMPath,
			WasmContentLengthHeader: h.Web.WasmContentLengthHeader,
			WorkerJS:                h.resolvePackagePath("/app-worker.js"),
			AutoUpdateInterval:      h.Web.AutoUpdateInterval.Milliseconds(),
		}); err != nil {
		panic(fmt.Errorf("initializing app.js failed: %w", err))
	}
	return b.Bytes()
}

func (h *builder) makeAppWorkerJS() []byte {
	resources := make(map[string]struct{})
	setResources := func(res ...string) {
		for _, r := range res {
			if r == "" {
				continue
			}

			r, _, _ = parseSrc(r)
			resources[r] = struct{}{}
		}
	}
	setResources(
		h.resolvePackagePath("/app.css"),
		h.resolvePackagePath("/app.js"),
		h.resolvePackagePath("/manifest.webmanifest"),
		h.resolvePackagePath("/wasm_exec.js"),
		h.resolvePackagePath("/"),
		appWASMPath,
	)
	// setResources(h.Icon.Default, h.Icon.Large, h.Icon.AppleTouch)
	// setResources(h.Styles...)
	// setResources(h.Fonts...)
	// setResources(h.Scripts...)
	// setResources(h.CacheableResources...)

	resourcesTocache := make([]string, 0, len(resources))
	for k := range resources {
		resourcesTocache = append(resourcesTocache, k)
	}
	sort.Slice(resourcesTocache, func(a, b int) bool {
		return strings.Compare(resourcesTocache[a], resourcesTocache[b]) < 0
	})

	var b bytes.Buffer
	if err := template.
		Must(template.New("app-worker.js").Parse(h.Web.ServiceWorkerTemplate)).
		Execute(&b, struct {
			Version          string
			ResourcesToCache string
		}{
			Version:          h.Version,
			ResourcesToCache: jsonString(resourcesTocache),
		}); err != nil {
		panic(fmt.Errorf("initializing app-worker.js failed: %w", err))
	}
	return b.Bytes()
}

func (h *builder) makeManifestJSON() []byte {
	normalize := func(s string) string {
		if !strings.HasPrefix(s, "/") {
			s = "/" + s
		}
		if !strings.HasSuffix(s, "/") {
			s += "/"
		}
		return s
	}

	var b bytes.Buffer
	if err := template.
		Must(template.New("manifest.webmanifest").Parse(manifestJSON)).
		Execute(&b, struct {
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
		}{
			// ShortName:       h.ShortName,
			Name:        h.Name,
			Description: h.Web.Description,
			// DefaultIcon:     h.Icon.Default,
			// LargeIcon:       h.Icon.Large,
			// SVGIcon:         h.Icon.SVG,
			BackgroundColor: h.Web.BackgroundColor,
			ThemeColor:      h.Web.ThemeColor,
			Scope:           normalize(packagePath),
			StartURL:        normalize(packagePath),
		}); err != nil {
		panic(fmt.Errorf("initializing manifest.webmanifest failed: %w", err))
	}
	return b.Bytes()
}

func (h *builder) makeIndexHTML() []byte {
	b := &bytes.Buffer{}
	err := template.Must(template.New("index.html").Parse(indexHTML)).Execute(b, nil)
	if err != nil {
		panic(fmt.Errorf("initializing index.html failed: %w", err))
	}
	return b.Bytes()
}

func (h *builder) initProxyResources() {
	// h.cachedResources = newMemoryCache(0)
	resources := make(map[string]ProxyResource)

	// for _, r := range h.ProxyResources {
	// 	switch r.Path {
	// 	case "/wasm_exec.js",
	// 		"/goapp.js",
	// 		"/app.js",
	// 		"/app-worker.js",
	// 		"/manifest.json",
	// 		"/manifest.webmanifest",
	// 		"/app.css",
	// 		"/app.wasm",
	// 		"/goapp.wasm",
	// 		"/":
	// 		continue

	// 	default:
	// 		if strings.HasPrefix(r.Path, "/") && strings.HasPrefix(r.ResourcePath, "/web/") {
	// 			resources[r.Path] = r
	// 		}
	// 	}
	// }

	if _, ok := resources["/robots.txt"]; !ok {
		resources["/robots.txt"] = ProxyResource{
			Path:         "/robots.txt",
			ResourcePath: "/web/robots.txt",
		}
	}
	if _, ok := resources["/sitemap.xml"]; !ok {
		resources["/sitemap.xml"] = ProxyResource{
			Path:         "/sitemap.xml",
			ResourcePath: "/web/sitemap.xml",
		}
	}
	if _, ok := resources["/ads.txt"]; !ok {
		resources["/ads.txt"] = ProxyResource{
			Path:         "/ads.txt",
			ResourcePath: "/web/ads.txt",
		}
	}

	// h.proxyResources = resources
}

// func (h *builder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	h.once.Do(h.init)

// 	w.Header().Set("Cache-Control", "no-cache")
// 	w.Header().Set("ETag", h.etag)

// 	etag := r.Header.Get("If-None-Match")
// 	if etag == h.etag {
// 		w.WriteHeader(http.StatusNotModified)
// 		return
// 	}

// 	path := r.URL.Path

// 	fileHandler, isServingStaticResources := h.Resources.(http.Handler)
// 	if isServingStaticResources && strings.HasPrefix(path, "/web/") {
// 		fileHandler.ServeHTTP(w, r)
// 		return
// 	}

// 	switch path {
// 	case "/goapp.js":
// 		path = "/app.js"

// 	case "/manifest.json":
// 		path = "/manifest.webmanifest"

// 	case "/app.wasm", "/goapp.wasm":
// 		if isServingStaticResources {
// 			r2 := *r
// 			r2.URL.Path = h.Resources.AppWASM()
// 			fileHandler.ServeHTTP(w, &r2)
// 			return
// 		}

// 		w.WriteHeader(http.StatusNotFound)
// 		return

// 	}

// 	if res, ok := h.cachedPWAResources.Get(path); ok {
// 		h.serveCachedItem(w, res)
// 		return
// 	}

// 	if proxyResource, ok := h.proxyResources[path]; ok {
// 		h.serveProxyResource(proxyResource, w, r)
// 		return
// 	}

// 	if library, ok := h.libraries[path]; ok {
// 		h.serveLibrary(w, r, library)
// 		return
// 	}

// 	h.servePage(w, r)
// }

// func (h *builder) serveCachedItem(w http.ResponseWriter, i cacheItem) {
// 	w.Header().Set("Content-Length", strconv.Itoa(i.Len()))
// 	w.Header().Set("Content-Type", i.ContentType)

// 	if i.ContentEncoding != "" {
// 		w.Header().Set("Content-Encoding", i.ContentEncoding)
// 	}

// 	w.WriteHeader(http.StatusOK)
// 	w.Write(i.Body)
// }

// func (h *builder) serveProxyResource(resource ProxyResource, w http.ResponseWriter, r *http.Request) {
// 	var u string
// 	if _, ok := h.Resources.(http.Handler); ok {
// 		var protocol string
// 		if r.TLS != nil {
// 			protocol = "https://"
// 		} else {
// 			protocol = "http://"
// 		}
// 		u = protocol + r.Host + resource.ResourcePath
// 	} else {
// 		u = h.Resources.Static() + resource.ResourcePath
// 	}

// 	if i, ok := h.cachedProxyResources.Get(resource.Path); ok {
// 		h.serveCachedItem(w, i)
// 		return
// 	}

// 	res, err := http.Get(u)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		Log(errors.New("getting proxy static resource failed").
// 			WithTag("url", u).
// 			WithTag("proxy-path", resource.Path).
// 			WithTag("static-resource-path", resource.ResourcePath).
// 			Wrap(err),
// 		)
// 		return
// 	}
// 	defer res.Body.Close()

// 	if res.StatusCode != http.StatusOK {
// 		w.WriteHeader(http.StatusNotFound)
// 		return
// 	}

// 	body, err := io.ReadAll(res.Body)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		Log(errors.New("reading proxy static resource failed").
// 			WithTag("url", u).
// 			WithTag("proxy-path", resource.Path).
// 			WithTag("static-resource-path", resource.ResourcePath).
// 			Wrap(err),
// 		)
// 		return
// 	}

// 	item := cacheItem{
// 		Path:            resource.Path,
// 		ContentType:     res.Header.Get("Content-Type"),
// 		ContentEncoding: res.Header.Get("Content-Encoding"),
// 		Body:            body,
// 	}
// 	h.cachedProxyResources.Set(item)
// 	h.serveCachedItem(w, item)
// }

// func (h *builder) servePage(w http.ResponseWriter, r *http.Request) {
// 	content, ok := routes.createComponent(r.URL.Path)
// 	if !ok {
// 		http.NotFound(w, r)
// 		return
// 	}

// 	url := *r.URL
// 	url.Host = r.Host
// 	url.Scheme = "http"

// 	page := requestPage{
// 		url:                   &url,
// 		resolveStaticResource: h.resolveStaticPath,
// 	}
// 	page.SetTitle(h.Title)
// 	page.SetLang(h.Lang)
// 	page.SetDescription(h.Description)
// 	page.SetAuthor(h.Author)
// 	page.SetKeywords(h.Keywords...)
// 	page.SetLoadingLabel(strings.ReplaceAll(h.LoadingLabel, "{progress}", "0"))
// 	page.SetImage(h.Image)

// 	disp := engine{
// 		Page:                   &page,
// 		IsServerSide:           true,
// 		StaticResourceResolver: h.resolveStaticPath,
// 		ActionHandlers:         actionHandlers,
// 	}
// 	body := h.Body().privateBody(
// 		Div(), // Pre-rendeging placeholder
// 		Aside().
// 			ID("app-wasm-loader").
// 			Class("goapp-app-info").
// 			Body(
// 				Img().
// 					ID("app-wasm-loader-icon").
// 					Class("goapp-logo goapp-spin").
// 					Src(h.Icon.Default),
// 				P().
// 					ID("app-wasm-loader-label").
// 					Class("goapp-label").
// 					Text(page.loadingLabel),
// 			),
// 	)
// 	if err := mount(&disp, body); err != nil {
// 		panic(errors.New("mounting pre-rendering container failed").
// 			WithTag("server-side", disp.isServerSide()).
// 			WithTag("body-type", reflect.TypeOf(disp.Body)).
// 			Wrap(err))
// 	}
// 	disp.Body = body
// 	disp.init()
// 	defer disp.Close()

// 	disp.Mount(content)

// 	for len(disp.dispatches) != 0 {
// 		disp.Consume()
// 		disp.Wait()
// 	}

// 	icon := h.Icon.SVG
// 	if icon == "" {
// 		icon = h.Icon.Default
// 	}

// 	var b bytes.Buffer
// 	b.WriteString("<!DOCTYPE html>\n")
// 	PrintHTML(&b, h.HTML().
// 		Lang(page.Lang()).
// 		privateBody(
// 			Head().Body(
// 				Meta().Charset("UTF-8"),
// 				Meta().
// 					Name("author").
// 					Content(page.Author()),
// 				Meta().
// 					Name("description").
// 					Content(page.Description()),
// 				Meta().
// 					Name("keywords").
// 					Content(page.Keywords()),
// 				Meta().
// 					Name("theme-color").
// 					Content(h.ThemeColor),
// 				Meta().
// 					Name("viewport").
// 					Content("width=device-width, initial-scale=1, maximum-scale=1, user-scalable=0, viewport-fit=cover"),
// 				Meta().
// 					Property("og:url").
// 					Content(page.URL().String()),
// 				Meta().
// 					Property("og:title").
// 					Content(page.Title()),
// 				Meta().
// 					Property("og:description").
// 					Content(page.Description()),
// 				Meta().
// 					Property("og:type").
// 					Content("website"),
// 				Meta().
// 					Property("og:image").
// 					Content(page.Image()),
// 				Range(page.twitterCardMap).Map(func(k string) UI {
// 					v := page.twitterCardMap[k]
// 					if v == "" {
// 						return nil
// 					}
// 					return Meta().
// 						Name(k).
// 						Content(v)
// 				}),
// 				Title().Text(page.Title()),
// 				Range(h.Preconnect).Slice(func(i int) UI {
// 					url, crossOrigin, _ := parseSrc(h.Preconnect[i])
// 					if url == "" {
// 						return nil
// 					}

// 					link := Link().
// 						Rel("preconnect").
// 						Href(url)

// 					if crossOrigin != "" {
// 						link = link.CrossOrigin(strings.Trim(crossOrigin, "true"))
// 					}

// 					return link
// 				}),
// 				Range(h.Fonts).Slice(func(i int) UI {
// 					url, crossOrigin, _ := parseSrc(h.Fonts[i])
// 					if url == "" {
// 						return nil
// 					}

// 					link := Link().
// 						Type("font/" + strings.TrimPrefix(filepath.Ext(url), ".")).
// 						Rel("preload").
// 						Href(url).
// 						As("font")

// 					if crossOrigin != "" {
// 						link = link.CrossOrigin(strings.Trim(crossOrigin, "true"))
// 					}

// 					return link
// 				}),
// 				Range(page.Preloads()).Slice(func(i int) UI {
// 					p := page.Preloads()[i]
// 					if p.Href == "" || p.As == "" {
// 						return nil
// 					}

// 					url, crossOrigin, _ := parseSrc(p.Href)
// 					if url == "" {
// 						return nil
// 					}

// 					link := Link().
// 						Type(p.Type).
// 						Rel("preload").
// 						Href(url).
// 						As(p.As).
// 						FetchPriority(p.FetchPriority)

// 					if crossOrigin != "" {
// 						link = link.CrossOrigin(strings.Trim(crossOrigin, "true"))
// 					}

// 					return link
// 				}),
// 				Range(h.Styles).Slice(func(i int) UI {
// 					url, crossOrigin, _ := parseSrc(h.Styles[i])
// 					if url == "" {
// 						return nil
// 					}

// 					link := Link().
// 						Type("text/css").
// 						Rel("preload").
// 						Href(url).
// 						As("style")

// 					if crossOrigin != "" {
// 						link = link.CrossOrigin(strings.Trim(crossOrigin, "true"))
// 					}

// 					return link
// 				}),
// 				Link().
// 					Rel("icon").
// 					Href(icon),
// 				Link().
// 					Rel("apple-touch-icon").
// 					Href(h.Icon.AppleTouch),
// 				Link().
// 					Rel("manifest").
// 					Href(h.resolvePackagePath("/manifest.webmanifest")),
// 				Range(h.Styles).Slice(func(i int) UI {
// 					url, crossOrigin, _ := parseSrc(h.Styles[i])
// 					if url == "" {
// 						return nil
// 					}

// 					link := Link().
// 						Rel("stylesheet").
// 						Type("text/css").
// 						Href(url)

// 					if crossOrigin != "" {
// 						link = link.CrossOrigin(strings.Trim(crossOrigin, "true"))
// 					}

// 					return link
// 				}),
// 				Script().
// 					Defer(true).
// 					Src(h.resolvePackagePath("/wasm_exec.js")),
// 				Script().
// 					Defer(true).
// 					Src(h.resolvePackagePath("/app.js")),
// 				Range(h.Scripts).Slice(func(i int) UI {
// 					url, crossOrigin, loading := parseSrc(h.Scripts[i])
// 					if url == "" {
// 						return nil
// 					}

// 					script := Script().Src(url)

// 					if crossOrigin != "" {
// 						script = script.CrossOrigin(strings.Trim(crossOrigin, "true"))
// 					}

// 					switch loading {
// 					case "defer":
// 						script = script.Defer(true)

// 					case "async":
// 						script.Async(true)
// 					}

// 					return script
// 				}),
// 				Range(h.RawHeaders).Slice(func(i int) UI {
// 					return Raw(h.RawHeaders[i])
// 				}),
// 			),
// 			body,
// 		))

// 	w.Header().Set("Content-Length", strconv.Itoa(b.Len()))
// 	w.Header().Set("Content-Type", "text/html")
// 	w.Write(b.Bytes())
// }

// func (h *builder) serveLibrary(w http.ResponseWriter, r *http.Request, library []byte) {
// 	w.Header().Set("Content-Length", strconv.Itoa(len(library)))
// 	w.Header().Set("Content-Type", "text/css")
// 	w.Write(library)
// }

func (h *builder) resolvePackagePath(path string) string {
	return strings.Trim(path, "/")
	// var b strings.Builder

	// b.WriteByte('/')
	// appResources := strings.Trim(h.Resources.Package(), "/")
	// b.WriteString(appResources)

	// path = strings.Trim(path, "/")
	// if b.Len() != 1 && path != "" {
	// 	b.WriteByte('/')
	// }
	// b.WriteString(path)

	// return b.String()
}

func (h *builder) resolveStaticPath(path string) string {
	return strings.Trim(path, "/")
	// if isRemoteLocation(path) || !isStaticResourcePath(path) {
	// 	return path
	// }

	// var b strings.Builder
	// staticResources := strings.TrimSuffix(h.Resources.Static(), "/")
	// b.WriteString(staticResources)
	// path = strings.Trim(path, "/")
	// b.WriteByte('/')
	// b.WriteString(path)
	// return b.String()
}

// // Icon describes a square image that is used in various places such as
// // application icon, favicon or loading icon.
// type Icon struct {
// 	// The path or url to a square image/png file. It must have a side of 192px.
// 	//
// 	// Path is relative to the root directory.
// 	Default string

// 	// The path or url to larger square image/png file. It must have a side of
// 	// 512px.
// 	//
// 	// Path is relative to the root directory.
// 	Large string

// 	// The path or url to a svg file.
// 	SVG string

// 	// The path or url to a square image/png file that is used for IOS/IPadOS
// 	// home screen icon. It must have a side of 192px.
// 	//
// 	// Path is relative to the root directory.
// 	//
// 	// DEFAULT: Icon.Default
// 	AppleTouch string
// }

// Environment describes the environment variables to pass to the progressive
// web app.
type Environment map[string]string

func isRemoteLocation(path string) bool {
	return strings.HasPrefix(path, "https://") ||
		strings.HasPrefix(path, "http://")
}

func isStaticResourcePath(path string) bool {
	return strings.HasPrefix(path, "/web/") ||
		strings.HasPrefix(path, "web/")
}

func parseSrc(link string) (url, crossOrigin, loading string) {
	for _, p := range strings.Split(link, " ") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		switch {
		case p == "crossorigin":
			crossOrigin = "true"

		case strings.HasPrefix(p, "crossorigin="):
			crossOrigin = strings.TrimPrefix(p, "crossorigin=")

		case p == "defer":
			loading = "defer"

		case p == "async":
			loading = "async"

		default:
			url = p
		}
	}

	return url, crossOrigin, loading
}
