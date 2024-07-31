// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build generatehtml

package gpu

import (
	"os"

	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// GLFWCreateWindow is a helper function intended only for use in simple examples that makes a
// new window with glfw on platforms that support it and is largely a no-op on other platforms.
func GLFWCreateWindow(gp *GPU, width, height int, title string) (surface *wgpu.Surface, terminate func(), pollEvents func() bool, err error) {
	os.Exit(0) // TODO: needs clean quit for generatehtml; is there any better solution?
	return
}