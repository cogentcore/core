// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package goscript provides support for running Cogent Core code compiled
// to JavaScript by [goscript] (https://github.com/s4wave/goscript), which is
// used for the js platform. Code compiled by goscript is built with
// GOOS=js, GOARCH=wasm, and the goscript build tag.
//
// The main use of this package is [Test], which runs the existing Go tests
// of a package compiled to JavaScript, so that the same tests verify both
// the native and the goscript versions of the code. Each package does this
// with a test file like this:
//
//	//go:build !goscript
//
//	package mypackage
//
//	import (
//		"testing"
//
//		"cogentcore.org/core/base/goscript"
//	)
//
//	func TestGoScript(t *testing.T) { goscript.Test(t) }
//
// This package must only import the standard library, so that it can be used
// in the tests of any package without an import cycle.
package goscript

import "testing"

// Skip skips the given test with the given reason if the code is running
// as JavaScript compiled by goscript (see [Enabled]). It is for the
// (ideally rare and temporary) tests that do not work in goscript.
func Skip(t testing.TB, reason string) {
	t.Helper()
	if Enabled {
		t.Skip("goscript: " + reason)
	}
}
