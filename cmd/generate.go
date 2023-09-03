// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import "goki.dev/goki/generate"

// GenerateCmd generates useful methods,
// variables, and constans for GoKi code.
func (a *App) GenerateCmd() error {
	return generate.Generate(a.Config())
}
