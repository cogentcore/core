// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import "goki.dev/goki/packman"

// ReleaseCmd releases the project
// by pushing a release with Git
// and releasing it on app stores
// if it is an executable app.
func (a *App) ReleaseCmd() error {
	return packman.Release(a.Config())
}
