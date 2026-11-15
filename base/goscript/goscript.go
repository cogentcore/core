// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package goscript provides support for running Cogent Core code compiled
// to JavaScript by [goscript] (https://github.com/s4wave/goscript), which is
// used for the js platform. Code compiled by goscript is built with
// GOOS=js, GOARCH=wasm, and the goscript build tag.
//
// The main use of this package is [RunTests], which runs the existing Go
// tests of a package compiled to JavaScript, so that the same tests verify
// both the native and the goscript versions of the code. It is run by
// core test js (see the cmd package), for the packages given to it, and,
// for a pattern with ... in it, for the packages that opt in with a
// [Directive] line in one of their test files:
//
//	//core:js-test
//
// Tests that do not work in goscript call [Skip].
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
