// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on http://github.com/dmarkham/enumer and
// golang.org/x/tools/cmd/stringer:

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumgen

import (
	"go/ast"

	"goki.dev/enums/enumgen/config"
)

// Type represents a parsed enum type.
type Type struct {
	Type      *ast.TypeSpec  // The standard AST type value
	IsBitFlag bool           // Whether the type is a bit flag type
	Extends   string         // The type that this type extends, if any ("" if it doesn't extend)
	Config    *config.Config // Configuration information set in the comment directive for the type; is initialized to generator config info first
}
