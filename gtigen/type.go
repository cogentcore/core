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
	Name           string         // The name of the type in its package (eg: MyType)
	FullName       string         // The fully package-path-qualified name of the type (eg: goki.dev/gi/v2.Button)
	ShortName      string         // The short, package-qualified name of the type (eg: gi.Button)
	IDName         string         // The short, package-unqualified, kebab-case name of the type, suitable for use in an ID (eg: button)
	Type           *ast.TypeSpec  // The standard AST type value
	Doc            string         // The documentation for the type
	Pkg            string         // The name of the package the type is in
	Directives     gti.Directives // The directives for the type; guaranteed to be non-nil
	Fields         *gti.Fields    // The fields of the struct type; nil if not a struct
	Embeds         *gti.Fields    // The embeds of the struct type; nil if not a struct
	EmbeddedFields *gti.Fields    // The fields contained within the embeds of the struct type; nil if not a struct, and used for generating setters only
	Methods        *gti.Methods   // The methods of the type; guaranteed to be non-nil
	Config         *Config        // Configuration information set in the comment directive for the type; is initialized to generator config info first
}
