// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gtigen

import (
	"go/ast"

	"goki.dev/gti"
)

// Type represents a parsed type.
type Type struct {
	Name       string         // The name of the type in its package (eg: MyType)
	FullName   string         // The fully-package-path-qualified name of the type (eg: goki.dev/ki/v2.MyType)
	Type       *ast.TypeSpec  // The standard AST type value
	Doc        string         // The documentation for the type
	Directives gti.Directives // The directives for the type; guaranteed to be non-nil
	Fields     *gti.Fields
	Config     *Config // Configuration information set in the comment directive for the type; is initialized to generator config info first
}

// Method represents a parsed method.
type Method struct {
	Name       string         // The name of the method
	Doc        string         // The documentation for the method
	Directives gti.Directives // The directives for the method; guaranteed to be non-nil
	Config     *Config        // Configuration information set in the comment directive for the method; is initialized to generator config info first
}
