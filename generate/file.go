// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generate

import (
	"go/ast"

	"goki.dev/goki/config"
)

// File holds a single parsed file and associated data.
type File struct {
	Pkg  *Package  // Package to which this file belongs.
	File *ast.File // Parsed AST.
	// These fields are reset for each type being generated.
	TypeName string         // The name of the constant type we are currently looking for.
	Config   *config.Config // The configuration information
}
