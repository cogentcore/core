// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !goscript

package video

import (
	"testing"

	"cogentcore.org/core/base/goscript"
)

// TestGoScript runs the tests of video_js_test.go in Chrome, which plays
// the H.264 and AAC of the test video (Chromium does not).
func TestGoScript(t *testing.T) {
	goscript.Test(t, goscript.Options{Browser: true, BrowserChannel: "chrome"})
}
