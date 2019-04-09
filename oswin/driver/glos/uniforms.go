// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build 3d

package glos

// uniform implements Uniform interface
type uniform struct {
	name   string
	array  bool
	ln     int32
	offset int32
	size   int32
	handle int32
}
