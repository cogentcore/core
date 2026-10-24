// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bytes"
	"compress/gzip"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/cmd/config"
)

// Serve serves the build output directory on the default network address at the config port.
// Text files and the app bundle are served compressed with gzip, as static
// site hosts do, so that the download sizes are the same as when deployed.
func Serve(c *config.Config) error {
	hfs := http.FileServer(http.Dir(c.Build.Output))
	gz := &gzipCache{}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		trim := strings.Trim(r.URL.Path, "/")
		status := http.StatusOK
		info, err := os.Stat(filepath.Join(c.Build.Output, trim))
		if err != nil {
			r.URL.Path = "/404.html"
			trim = "404.html"
			status = http.StatusNotFound
		}
		if trim == "app.wasm" {
			w.Header().Set("Content-Type", "application/wasm")
		}
		if c.Web.Gzip && slices.Contains(bundleFiles(c), trim) {
			// The bundle files are already compressed (see [processBundle]).
			// The file server does not set the Content-Length of an encoded
			// response, which the app loader uses to show progress.
			w.Header().Set("Content-Encoding", "gzip")
			if r.Header.Get("Range") == "" {
				w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
			}
		} else if serveGzip(w, r, gz, c.Build.Output, trim, status) {
			return
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
		}
		hfs.ServeHTTP(w, r)
	})

	logx.PrintlnWarn("Serving at http://localhost:" + c.Web.Port)
	return http.ListenAndServe(":"+c.Web.Port, nil)
}

// gzipExts are the extensions of the files that [serveGzip] compresses.
var gzipExts = []string{".html", ".js", ".css", ".json", ".wasm", ".webmanifest", ".svg", ".txt", ".md"}

// serveGzip serves the file at the given path relative to the given
// directory compressed with gzip with the given status, if the request
// accepts gzip and the file is one that compresses well (see [gzipExts]).
// The path of a directory serves its index.html. It returns whether it
// served the file.
func serveGzip(w http.ResponseWriter, r *http.Request, gz *gzipCache, dir, name string, status int) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	// Ranges are of the uncompressed file, which the file server handles.
	if !acceptsGzip(r.Header.Get("Accept-Encoding")) || r.Header.Get("Range") != "" {
		return false
	}
	fn := filepath.Join(dir, filepath.FromSlash(name))
	info, err := os.Stat(fn)
	if err != nil {
		return false
	}
	if info.IsDir() {
		// The file server redirects a directory without a trailing slash.
		if name != "" && !strings.HasSuffix(r.URL.Path, "/") {
			return false
		}
		fn = filepath.Join(fn, "index.html")
		info, err = os.Stat(fn)
		if err != nil {
			return false
		}
	}
	ext := path.Ext(fn)
	if !slices.Contains(gzipExts, ext) {
		return false
	}
	b, err := gz.get(fn, info)
	if err != nil {
		return false
	}
	h := w.Header()
	if h.Get("Content-Type") == "" {
		ct := mime.TypeByExtension(ext)
		if ct == "" {
			ct = "application/octet-stream"
		}
		h.Set("Content-Type", ct)
	}
	h.Set("Content-Encoding", "gzip")
	h.Add("Vary", "Accept-Encoding")
	// The app loader shows download progress with the Content-Length, which
	// ServeContent does not set for an encoded response.
	h.Set("Content-Length", strconv.Itoa(len(b)))
	if status != http.StatusOK {
		w.WriteHeader(status)
		if r.Method != http.MethodHead {
			w.Write(b)
		}
		return true
	}
	http.ServeContent(w, r, fn, info.ModTime(), bytes.NewReader(b))
	return true
}

// gzipCache is a cache of the files compressed by [serveGzip], so that each
// version of a file is only compressed once.
type gzipCache struct {
	mu    sync.Mutex
	files map[string]*gzipFile
}

// gzipFile is a file compressed with gzip in a [gzipCache].
type gzipFile struct {

	// ModTime and Size are those of the file that was compressed.
	ModTime time.Time
	Size    int64

	// Data is the compressed file.
	Data []byte
}

// get returns the compressed contents of the file with the given name
// and info, compressing it if it is not in the cache.
func (gc *gzipCache) get(name string, info os.FileInfo) ([]byte, error) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	if f, ok := gc.files[name]; ok && f.ModTime.Equal(info.ModTime()) && f.Size == info.Size() {
		return f.Data, nil
	}
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	gz, err := gzipBytes(b, gzip.DefaultCompression)
	if err != nil {
		return nil, err
	}
	if gc.files == nil {
		gc.files = map[string]*gzipFile{}
	}
	gc.files[name] = &gzipFile{ModTime: info.ModTime(), Size: info.Size(), Data: gz}
	return gz, nil
}
