// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"net/http"
	"path/filepath"

	"cogentcore.org/core/core/config"
	"cogentcore.org/core/grog"
)

// Serve serves the build output directory on the default network address at the config port.
func Serve(c *config.Config) error {
	fs := http.FileServer(http.Dir(filepath.Dir(c.Build.Output)))
	http.Handle("/", fs)
	http.HandleFunc("/app.wasm", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/wasm")
		if c.Web.Gzip {
			w.Header().Set("Content-Encoding", "gzip")
		}
		fs.ServeHTTP(w, r)
	})

	grog.PrintlnWarn("Serving at http://localhost:" + c.Web.Port)
	return http.ListenAndServe(":"+c.Web.Port, nil)
}
