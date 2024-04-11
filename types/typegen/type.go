// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package typegen

import (
	"go/ast"

	"cogentcore.org/core/types"
)

// Type represents a parsed type.
type Type struct {
	types.Type

	// LocalName is the name of the type in its package
	LocalName string

	// The standard AST type value
	AST *ast.TypeSpec

	// The name of the package the type is in
	Pkg string

	// The fields of the struct type; nil if not a struct
	Fields Fields

	// The embeds of the struct type; nil if not a struct
	Embeds Fields

	// The fields contained within the embeds of the struct type;
	// nil if not a struct, and used for generating setters only
	EmbeddedFields Fields

	// Configuration information set in the comment directive for the type;
	// is initialized to generator config info first
	Config *Config
}

// Fields extends []gti.Field to provide the local type names and struct tags for each field.
type Fields struct {
	Fields     []types.Field
	LocalTypes map[string]string
	Tags       map[string]string
}
