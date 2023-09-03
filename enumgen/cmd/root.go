// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import "goki.dev/enums/enumgen"

// RootCmd is the root command of enumgen
// that generates the enum methods
func (a *App) RootCmd() error {
	return enumgen.Generate(a.Config())
}
