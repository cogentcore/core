// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generate

import (
	"go/ast"

	"goki.dev/goki/config"
)

// Type represents a parsed Ki type.
type Type struct {
	Type      *ast.TypeSpec  // The standard AST type value
	IsBitFlag bool           // Whether the type is a bit flag type
	Config    *config.Config // Configuration information set in the comment directive for the type; is initialized to generator config info first
}
