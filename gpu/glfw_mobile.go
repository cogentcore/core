// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android || ios

package gpu

// GLFWPollEvents is a wrapper around glfw.PollEvents that is a no-op on mobile
func GLFWPollEvents() {
}
