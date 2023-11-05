// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"net/http"
	"path/filepath"

	"goki.dev/goki/config"
)

// Serve serves the build output directory on localhost:8080.
func Serve(c *config.Config) error {
	fs := http.FileServer(http.Dir(filepath.Dir(c.Build.Output)))
	http.Handle("/", fs)

	return http.ListenAndServe(":8080", nil)
}
