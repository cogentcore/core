// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"reflect"
)

// This file contains helpful functions for dealing with slices, in the reflect
// system

// SliceElType returns the type of the elements for the given slice (which can be
// a pointer to a slice or a direct slice) -- just Elem() of slice type, but using
// this function makes it more explicit what is going on.
func SliceElType(sl interface{}) reflect.Type {
	return NonPtrType(reflect.TypeOf(sl)).Elem()
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func SliceNewAt(sl interface{}, idx int) {
	sltyp := SliceElType(sl)
	slptr := sltyp.Kind() == reflect.Ptr

	svl := reflect.ValueOf(sl)
	svnp := NonPtrValue(svl)

	nval := reflect.New(NonPtrType(sltyp)) // make the concrete el
	if !slptr {
		nval = nval.Elem() // use concrete value
	}
	sz := svnp.Len()
	svnp = reflect.Append(svnp, nval)
	if idx >= 0 && idx < sz {
		reflect.Copy(svnp.Slice(idx+1, sz+1), svnp.Slice(idx, sz))
		svnp.Index(idx).Set(nval)
	}
	svl.Elem().Set(svnp)
}

// SliceDeleteAt deletes element at given index from slice
func SliceDeleteAt(sl interface{}, idx int) {
	svl := reflect.ValueOf(sl)
	svnp := NonPtrValue(svl)
	svtyp := svnp.Type()
	nval := reflect.New(svtyp.Elem())
	sz := svnp.Len()
	reflect.Copy(svnp.Slice(idx, sz-1), svnp.Slice(idx+1, sz))
	svnp.Index(sz - 1).Set(nval.Elem())
	svl.Elem().Set(svnp.Slice(0, sz-1))
}
