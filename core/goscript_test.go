// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !goscript

package core

import (
	"testing"

	"cogentcore.org/core/base/goscript"
)

// TestGoScript runs the tests of rendering that differs on the web, compiled
// to JavaScript by goscript, in a browser (see [goscript.Test]).
func TestGoScript(t *testing.T) {
	goscript.Test(t, goscript.Options{Browser: true, BrowserChannel: "chrome", Run: "^TestDragImage$"})
}
