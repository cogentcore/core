// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"fmt"
	"net/http"
	"strings"

	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/cmd/core/config"
)

// Serve serves the build output directory on the default network address at the config port.
func Serve(c *config.Config) error {
	fs := http.FileServer(http.Dir(c.Build.Output))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		trim := strings.Trim(r.URL.Path, "/")
		fmt.Println(trim)
		if trim == "app.wasm" {
			w.Header().Set("Content-Type", "application/wasm")
			if c.Web.Gzip {
				w.Header().Set("Content-Encoding", "gzip")
			}
		}
		fs.ServeHTTP(w, r)
	})

	logx.PrintlnWarn("Serving at http://localhost:" + c.Web.Port)
	return http.ListenAndServe(":"+c.Web.Port, nil)
}
