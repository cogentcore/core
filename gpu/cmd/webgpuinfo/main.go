// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"cogentcore.org/core/base/reflectx"
	"github.com/cogentcore/webgpu/wgpu"
)

func main() {
	instance := wgpu.CreateInstance(nil)

	gpus := instance.EnumerateAdapters(nil)
	for i, a := range gpus {
		props := a.GetProperties()
		fmt.Println("\n########################\nWebGPU Adapter number:", i)
		fmt.Println(reflectx.StringJSON(props))
		limits := a.GetLimits()
		fmt.Println(reflectx.StringJSON(limits))
	}
}