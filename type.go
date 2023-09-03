// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"
	"reflect"
	"sync/atomic"
)

var (
	// TypeIDCounter is an atomically incremented uint64 used
	// for assigning new [Type.ID] numbers
	TypeIDCounter uint64

	// TypeRegistry provides a way to look up types from string short names (package.Type, e.g., gi.Button)
	TypeRegistry = map[string]*Type{}
)

// Type represents a Ki type
type Type struct {

	// type name, using the short form (e.g., gi.Button)
	Name string `desc:"type name, using the short form (e.g., gi.Button)"`

	// unique type ID number; use as a key handle for embed map
	ID uint64 `desc:"unique type ID number; use as a key handle for embed map"`

	// instance of the Ki type -- call Clone() on this to make a new token
	Instance Ki `desc:"instance of the Ki type -- call Clone() on this to make a new token"`
}

// NewType creates a new Type for given instance. This call is auto-generated for each Ki type.
func NewType(nm string, inst Ki) *Type {
	inst.InitName(inst, nm)
	tp := &Type{Name: nm, Instance: inst}
	tp.ID = atomic.AddUint64(&TypeIDCounter, 1)
	TypeRegistry[nm] = tp
	return tp
}

// NewInstance returns a new instance of given type
// Note: otherwise impossible to generate new instance generically, unless using reflection
func (tp *Type) NewInstance() Ki {
	return tp.Instance.NewInstance()
}

// ReflectType returns the [reflect.Type] of a given Ki Type
func (tp *Type) ReflectType() reflect.Type {
	return reflect.TypeOf(tp.Instance).Elem()
}

// TypeByName returns a ki Type by name (package.Type, e,g. gi.Button), or error if not found
func TypeByName(nm string) (*Type, error) {
	tp, ok := TypeRegistry[nm]
	if !ok {
		return nil, fmt.Errorf("Ki Type: %s not found", nm)
	}
	return tp, nil
}
