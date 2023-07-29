// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

//go:build 386 || arm
// +build 386 arm

package vgpu

// ByteCopyMemoryLimit represents the total number of bytes
// that can be copied from a Vulkan Memory Buffer to a byte slice.
const ByteCopyMemoryLimit int = 0x7FFFFFF
