// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packman

import (
	"path/filepath"

	"goki.dev/goki/config"
	"goki.dev/goki/web"
)

// Serve builds the package into static web files and then
// serves them on localhost at the config port.
func Serve(c *config.Config) error { //gti:add
	// needed for changes to show during local development
	c.Web.RandomVersion = true
	// need to get real output location so that commands work
	if c.Build.Output == "" {
		c.Build.Output = filepath.Join(".goki", "web", "app.wasm")
	}
	err := web.Build(c)
	if err != nil {
		return err
	}
	return web.Serve(c)
}
