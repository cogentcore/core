// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metadata

import (
	"fmt"
	"maps"
)

// Data is metadata as a map of named any elements
// with generic support for type-safe Get and nil-safe Set.
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

// Get gets metadata value of given type.
// returns error if not present or item is a different type.
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
