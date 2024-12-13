// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build generatehtml

package gpu

import (
	"image"
	"os"

	"github.com/cogentcore/webgpu/wgpu"
)

func init() {
	// Needs immediate clean quit for generatehtml;
	// otherwise, the app will hang forever.
	os.Exit(0)
}

// GLFWCreateWindow is a helper function intended only for use in simple examples that makes a
// new window with glfw on platforms that support it and is largely a no-op on other platforms.
func GLFWCreateWindow(size image.Point, title string, resize *func(size image.Point)) (surface *wgpu.Surface, terminate func(), pollEvents func() bool, actualSize image.Point, err error) {
	return
}
