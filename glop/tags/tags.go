// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tags provides support for advanced struct tags defined
// using a TypeTags method.
package tags

import "reflect"

// TypeTagger is the interface that types can implement to define
// advanced struct tags in code.
type TypeTagger interface {

	// TypeTags returns the advanced struct tags for the fields
	// of this type. The first map is keyed by field name and the
	// second map is keyed by tag name.
	TypeTags() map[string]map[string]any
}

// Get looks for the given tag on the given field for the given value.
// If it implements the [TypeTagger] interface, it uses that. Otherwise, or
// if it can not find it in [TypeTagger.TypeTags], it uses standard reflect
// struct tags. It returns the resulting value and nil if it was not found.
// See [Lookup] for a version that returns whether it was found.
func Get(v any, field string, tag string) any {
	res, _ := Lookup(v, field, tag)
	return res
}

// Lookup looks for the given tag on the given field for the given value.
// If it implements the [TypeTagger] interface, it uses that. Otherwise, or
// if it can not find it in [TypeTagger.TypeTags], it uses standard reflect
// struct tags. It returns the resulting value and whether it was found.
// See [Get] for a version that does not return whether it was found.
func Lookup(v any, field string, tag string) (any, bool) {
	if tt, ok := v.(TypeTagger); ok {
		ftags, ok := tt.TypeTags()[field]
		if !ok {
			goto usingReflect
		}
		res, ok := ftags[tag]
		if !ok {
			goto usingReflect
		}
		return res, true
	}
usingReflect:
	rt := reflect.TypeOf(v)
	// must be non-pointer, and we can't use laser.NonPtrType due to import cycle
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	sf, ok := rt.FieldByName(field)
	if !ok {
		return nil, false
	}
	res, ok := sf.Tag.Lookup(tag)
	if !ok {
		return nil, false
	}
	return res, true
}
