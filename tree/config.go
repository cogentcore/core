// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"fmt"

	"cogentcore.org/core/gti"
)

// TypeAndName holds a type and a name. It is used for specifying [Config]
// objects in [Node.ConfigChildren].
type TypeAndName struct {
	Type *gti.Type
	Name string
}

// Config is a list of [TypeAndName]s used in [Node.ConfigChildren].
type Config []TypeAndName

// Add adds a new configuration entry with the given type and name.
func (t *Config) Add(typ *gti.Type, name string) {
	*t = append(*t, TypeAndName{typ, name})
}

func (t Config) GoString() string {
	var str string
	for i, tn := range t {
		str += fmt.Sprintf("[%02d: %20s\t %20s\n", i, tn.Name, tn.Type.Name)
	}
	return str
}
