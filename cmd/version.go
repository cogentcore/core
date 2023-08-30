// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import "goki.dev/goki/packman"

// VersionCmd updates the version file
// of the project and commits and pushes
// the changes.
func (a *App) VersionCmd() error {
	return packman.UpdateVersion(a.Config())
}
