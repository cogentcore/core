// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"reflect"
	"unsafe"
)

// IsNil is a type-robust nil checker for vulkan nondispatchable handles
// which can be either uint64 or pointers.
func IsNil(ptr any) bool {
	if ui, ok := ptr.(uint64); ok {
		return ui == 0
	}
	val := reflect.ValueOf(ptr)
	return val.IsNil()
}

// SetNil is a type-robust nil checker for vulkan nondispatchable handles
// which can be either uint64 or pointers.
func SetNil(ptr unsafe.Pointer) {
	*(**int)(ptr) = nil
}
