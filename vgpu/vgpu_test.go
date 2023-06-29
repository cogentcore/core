// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"testing"
	"unsafe"

	vk "github.com/goki/vulkan"
)

// TestPtrFuncs can only be run on desktop platform where actual pointers are used
func TestPtrFuncs(t *testing.T) {
	var ptr32bit uint64
	var cmdPool vk.CommandPool

	if !IsNil(ptr32bit) {
		t.Errorf("ptr32bit should be nil!\n")
	}
	if !IsNil(cmdPool) {
		t.Errorf("cmdPool should be nil!\n")
	}

	ptr32bit = 10
	cmdPool = vk.CommandPool(unsafe.Add(unsafe.Pointer(cmdPool), 100))

	if IsNil(ptr32bit) {
		t.Errorf("ptr32bit should not be nil!\n")
	}
	if IsNil(cmdPool) {
		t.Errorf("cmdPool should not be nil!\n")
	}

	SetNil(unsafe.Pointer(&ptr32bit))
	SetNil(unsafe.Pointer(&cmdPool))

	if !IsNil(ptr32bit) {
		t.Errorf("ptr32bit should be nil!\n")
	}
	if !IsNil(cmdPool) {
		t.Errorf("cmdPool should be nil!\n")
	}

}
