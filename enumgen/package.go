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
