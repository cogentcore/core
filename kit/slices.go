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
