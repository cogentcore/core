// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !goscript

package core

import (
	"testing"
	"time"

	"cogentcore.org/core/base/goscript"
)

// TestGoScript runs the tests compiled to JavaScript by goscript, in a
// browser (see [goscript.Test]): on the web, which has one window, the size
// of a desktop window (see viewport_js_test.go), with its own test images
// in testdata/js, and the tests that do not apply there skipped (see
// skipOnWeb).
func TestGoScript(t *testing.T) {
	goscript.Test(t, goscript.Options{Browser: true, BrowserChannel: "chrome", Timeout: 20 * time.Minute})
}
