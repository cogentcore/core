// Copyright (c) 2019, The GoKi Authors. All rights reserved.
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

// CompareAndSwapFloat32 executes the atomic compare-and-swap operation
// returning true on success, using Uint32 and bit conversions.
func CompareAndSwapFloat32(addr *float32, old, nw float32) (swapped bool) {
	return atomic.CompareAndSwapUint32(
		(*uint32)(unsafe.Pointer(addr)),
		math.Float32bits(old),
		math.Float32bits(nw),
	)
}

// LoadFloat32 executes the atomic Load operation, using Uint32 and bit conversions.
func LoadFloat32(addr *float32) float32 {
	return math.Float32frombits(atomic.LoadUint32((*uint32)(unsafe.Pointer(addr))))
}

// StoreFloat32 executes the atomic Store operation, using Uint32 and bit conversions.
func StoreFloat32(addr *float32, val float32) {
	atomic.StoreUint32((*uint32)(unsafe.Pointer(addr)), math.Float32bits(val))
}

// SwapFloat32 executes the atomic Swap operation, using Uint32 and bit conversions.
func SwapFloat32(addr *float32, nw float32) float32 {
	return math.Float32frombits(atomic.SwapUint32((*uint32)(unsafe.Pointer(addr)), math.Float32bits(nw)))
}

// AddFloat32 adds given increment to a float32 value atomically
func AddFloat32(addr *float32, inc float32) (nval float32) {
	for {
		old := *addr
		nval = old + inc
		if CompareAndSwapFloat32(addr, old, nval) {
			break
		}
	}
	return
}

// CompareAndSwapFloat64 executes the compare-and-swap operation
// returning true on success, using Uint64 and bit conversions.
func CompareAndSwapFloat64(addr *float64, old, nw float64) (swapped bool) {
	return atomic.CompareAndSwapUint64(
		(*uint64)(unsafe.Pointer(addr)),
		math.Float64bits(old),
		math.Float64bits(nw),
	)
}

// LoadFloat64 executes the atomic Load operation, using Uint64 and bit conversions.
func LoadFloat64(addr *float64) float64 {
	return math.Float64frombits(atomic.LoadUint64((*uint64)(unsafe.Pointer(addr))))
}

// StoreFloat64 executes the atomic Store operation, using Uint64 and bit conversions.
func StoreFloat64(addr *float64, val float64) {
	atomic.StoreUint64((*uint64)(unsafe.Pointer(addr)), math.Float64bits(val))
}

// SwapFloat64 executes the atomic Swap operation, using Uint64 and bit conversions.
func SwapFloat64(addr *float64, nw float64) float64 {
	return math.Float64frombits(atomic.SwapUint64((*uint64)(unsafe.Pointer(addr)), math.Float64bits(nw)))
}

// AddFloat64 adds given increment to a float64 value atomically
func AddFloat64(addr *float64, inc float64) (nval float64) {
	for {
		old := *addr
		nval = old + inc
		if CompareAndSwapFloat64(addr, old, nval) {
			break
		}
	}
	return
}
