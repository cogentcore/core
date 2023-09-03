// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generate

import (
	"go/ast"
	"go/types"
)

// Package holds information about a Go package
type Package struct {
	Dir      string
	Name     string
	Defs     map[*ast.Ident]types.Object
	Files    []*File
	TypesPkg *types.Package
}
