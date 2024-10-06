// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/cli"
	"cogentcore.org/core/goal/gosl/gotosl"
)

func main() { //types:skip
	opts := cli.DefaultOptions("gosl", "Go as a shader language converts Go code to WGSL WebGPU shader code, which can be run on the GPU through WebGPU.")
	cfg := &gotosl.Config{}
	cli.Run(opts, cfg, gotosl.Run)
}
