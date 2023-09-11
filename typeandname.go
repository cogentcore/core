// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"goki.dev/gti"
)

// TypeAndName holds a type and a name.
// Used for specifying configurations of children in Ki,
// for efficiently configuring the chilren.
type TypeAndName struct {
	Type *gti.Type
	Name string
}

// list of type-and-names -- can be created from a string spec
type TypeAndNameList []TypeAndName

func (t *TypeAndNameList) Add(typ *gti.Type, nm string) {
	(*t) = append(*t, TypeAndName{typ, nm})
}
