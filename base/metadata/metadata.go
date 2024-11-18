// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package metadata provides a map of named any elements
// with generic support for type-safe Get and nil-safe Set.
// Metadata keys often function as optional fields in a struct,
// and therefore a CamelCase naming convention is typical.
// Provides default support for "Name", "Doc", "File" standard keys.
package metadata

import (
	"fmt"
	"maps"

	"cogentcore.org/core/base/errors"
)

// Data is metadata as a map of named any elements
// with generic support for type-safe Get and nil-safe Set.
// Metadata keys often function as optional fields in a struct,
// and therefore a CamelCase naming convention is typical.
// Provides default support for "Name" and "Doc" standard keys.
// In general it is good practice to provide access functions
// that establish standard key names, to avoid issues with typos.
type Data map[string]any

func (md *Data) init() {
	if *md == nil {
		*md = make(map[string]any)
	}
}

// Set sets key to given value, ensuring that
// the map is created if not previously.
func (md *Data) Set(key string, value any) {
	md.init()
	(*md)[key] = value
}

// Get gets metadata value of given type from given Data.
// Returns error if not present or item is a different type.
func Get[T any](md Data, key string) (T, error) {
	var z T
	x, ok := md[key]
	if !ok {
		return z, fmt.Errorf("key %q not found in metadata", key)
	}
	v, ok := x.(T)
	if !ok {
		return z, fmt.Errorf("key %q has a different type than expected %T: is %T", key, z, x)
	}
	return v, nil
}

// Copy does a shallow copy of metadata from source.
// Any pointer-based values will still point to the same
// underlying data as the source, but the two maps remain
// distinct.  It uses [maps.Copy].
func (md *Data) Copy(src Data) {
	if src == nil {
		return
	}
	md.init()
	maps.Copy(*md, src)
}

//////// Metadataer

// Metadataer is an interface for a type that returns associated
// metadata.Data using a Metadata() method. To be able to set metadata,
// the method should be defined with a pointer receiver.
type Metadataer interface {
	Metadata() *Data
}

// GetData gets the Data from given object, if it implements the
// Metadata() method. Returns nil if it does not.
// Must pass a pointer to the object.
func GetData(obj any) *Data {
	if md, ok := obj.(Metadataer); ok {
		return md.Metadata()
	}
	return nil
}

// GetFrom gets metadata value of given type from given object,
// if it implements the Metadata() method.
// Must pass a pointer to the object.
// Returns error if not present or item is a different type.
func GetFrom[T any](obj any, key string) (T, error) {
	md := GetData(obj)
	if md == nil {
		var zv T
		return zv, errors.New("metadata not available for given object type")
	}
	return Get[T](*md, key)
}

// SetTo sets metadata value on given object, if it implements
// the Metadata() method. Returns error if no Metadata on object.
// Must pass a pointer to the object.
func SetTo(obj any, key string, value any) error {
	md := GetData(obj)
	if md == nil {
		return errors.Log(errors.New("metadata not available for given object type"))
	}
	md.Set(key, value)
	return nil
}
