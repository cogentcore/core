// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateStaticWebsite generates the files to run a PWA built with go-app as a
// static website in the specified directory. Static websites can be used with
// hosts such as Github Pages.
//
// Note that app.wasm must still be built separately and put into the web
// directory.
func GenerateStaticWebsite(dir string, b *builder, pages ...string) error {
	if dir == "" {
		dir = "."
	}

	resources := map[string]struct{}{
		"/":                     {},
		"/wasm_exec.js":         {},
		"/app.js":               {},
		"/app-worker.js":        {},
		"/manifest.webmanifest": {},
		"/app.css":              {},
		"/web":                  {},
	}

	for _, path := range pages {
		resources[path] = struct{}{}
	}

	for _, p := range pages {
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		resources[p] = struct{}{}
	}

	// server := httptest.NewServer(h)
	// defer server.Close()

	for path := range resources {
		switch path {
		case "/web":
			if err := createStaticDir(filepath.Join(dir, path), ""); err != nil {
				return fmt.Errorf("creating web directory failed: %w", err)
			}

		default:
			filename := path
			if filename == "/" {
				filename = "/index.html"
			}

			f, err := createStaticFile(dir, filename)
			if err != nil {
				return fmt.Errorf("creating file failed: path=%v filename=%v: %w", path, filename, err)
			}
			defer f.Close()

			page, err := createStaticPage(b, "/"+path)
			if err != nil {
				return fmt.Errorf("creating page failed: path=%v filename=%v: %w", path, filename, err)
			}

			if n, err := f.Write(page); err != nil {
				return fmt.Errorf("writing page failed: path=%v filename=%v bytes-written=%v: %w", path, filename, n, err)
			}
		}
	}

	return nil
}

func createStaticDir(dir, path string) error {
	dir = filepath.Join(dir, filepath.Dir(path))
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return nil
	}
	return os.MkdirAll(filepath.Join(dir), 0755)
}

func createStaticFile(dir, path string) (*os.File, error) {
	if err := createStaticDir(dir, path); err != nil {
		return nil, fmt.Errorf("creating file directory failed: %w", err)
	}

	filename := filepath.Join(dir, path)
	if filepath.Ext(filename) == "" {
		filename += ".html"
	}

	return os.Create(filename)
}

func createStaticPage(b *builder, path string) ([]byte, error) {
	item, ok := b.cachedResources.Get(path)
	if !ok {
		return nil, fmt.Errorf("path not in cache: %q", path)
	}
	return item.Body, nil
}
