// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packman

import "goki.dev/goki/config"

// Release releases the config app/library
// by calling [ReleaseApp] if it is an app
// and [ReleaseLibrary] if it is a library.
func Release(c *config.Config) error {
	if c.Type == config.TypeApp {
		return ReleaseApp(c)
	}
	return ReleaseLibrary(c)
}

// ReleaseApp releases the config app.
func ReleaseApp(c *config.Config) error {
	// TODO: implement
	return nil
}

// ReleaseLibrary releases the config library.
func ReleaseLibrary(c *config.Config) error {
	// TODO: implement
	return nil
}
