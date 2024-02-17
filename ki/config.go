// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"

	"cogentcore.org/core/gti"
)

// TypeAndName holds a type and a name.
// Used for specifying configurations of children in Ki,
// for efficiently configuring the chilren.
type TypeAndName struct {
	Type *gti.Type
	Name string
}

// Config list of type-and-names -- can be created from a string spec
type Config []TypeAndName

func (t *Config) Add(typ *gti.Type, nm string) {
	*t = append(*t, TypeAndName{typ, nm})
}

func (t Config) GoString() string {
	var str string
	for i, tn := range t {
		str += fmt.Sprintf("[%02d: %20s\t %20s\n", i, tn.Name, tn.Type.Name)
	}
	return str
}
