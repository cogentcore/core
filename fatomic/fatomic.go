// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fatomic provides floating-point atomic operations
package fatomic

import (
	"math"
	"sync/atomic"
	"unsafe"
)

// from:
// https://stackoverflow.com/questions/27492349/go-atomic-addfloat32

// AddFloat32 adds given increment to a float32 value atomically
func AddFloat32(val *float32, inc float32) (nval float32) {
	for {
		old := *val
		nval = old + inc
		if atomic.CompareAndSwapUint32(
			(*uint32)(unsafe.Pointer(val)),
			math.Float32bits(old),
			math.Float32bits(nval),
		) {
			break
		}
	}
	return
}

// AddFloat64 adds given increment to a float64 value atomically
func AddFloat64(val *float64, inc float64) (nval float64) {
	for {
		old := *val
		nval = old + inc
		if atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(val)),
			math.Float64bits(old),
			math.Float64bits(nval),
		) {
			break
		}
	}
	return
}
