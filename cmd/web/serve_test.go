// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAcceptsGzip(t *testing.T) {
	assert.True(t, acceptsGzip("gzip, deflate, br, zstd"))
	assert.True(t, acceptsGzip("br;q=1.0, gzip;q=0.8"))
	assert.True(t, acceptsGzip("*"))
	assert.False(t, acceptsGzip(""))
	assert.False(t, acceptsGzip("br"))
	assert.False(t, acceptsGzip("gzip;q=0"))
	assert.False(t, acceptsGzip("gzip; q=0.0"))
}

func TestServeGzip(t *testing.T) {
	dir := t.TempDir()
	js := strings.Repeat("console.log('hello');\n", 1000)
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "app.gs.js"), []byte(js), 0666))
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "icon.png"), []byte("png"), 0666))
	assert.NoError(t, os.MkdirAll(filepath.Join(dir, "page"), 0777))
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "page", "index.html"), []byte("<p>page</p>"), 0666))
	gz := &gzipCache{}

	get := func(name, path, accept string) (*httptest.ResponseRecorder, bool) {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		if accept != "" {
			r.Header.Set("Accept-Encoding", accept)
		}
		w := httptest.NewRecorder()
		return w, serveGzip(w, r, gz, dir, name, http.StatusOK)
	}
	unzip := func(w *httptest.ResponseRecorder) string {
		zr, err := gzip.NewReader(w.Body)
		assert.NoError(t, err)
		b, err := io.ReadAll(zr)
		assert.NoError(t, err)
		return string(b)
	}

	w, ok := get("app.gs.js", "/app.gs.js", "gzip, br")
	assert.True(t, ok)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	assert.Contains(t, w.Header().Get("Content-Type"), "javascript")
	assert.Less(t, w.Body.Len(), len(js)/10)
	assert.Equal(t, strconv.Itoa(w.Body.Len()), w.Header().Get("Content-Length"))
	assert.Equal(t, js, unzip(w))

	_, ok = get("app.gs.js", "/app.gs.js", "")
	assert.False(t, ok)
	_, ok = get("icon.png", "/icon.png", "gzip")
	assert.False(t, ok)

	w, ok = get("page", "/page/", "gzip")
	assert.True(t, ok)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Equal(t, "<p>page</p>", unzip(w))
	_, ok = get("page", "/page", "gzip") // redirected by the file server
	assert.False(t, ok)
}
