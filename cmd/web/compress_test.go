// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStartupFiles(t *testing.T) {
	// As esbuild writes them: the entry imports a shared chunk statically
	// and the chunk of a deferred function dynamically, which imports the
	// shared chunk and another chunk of its own statically.
	files := []string{"app.gs.js", "app.gs.chunk-shared.js", "app.gs.chunk-deferred.js", "app.gs.chunk-deferred2.js"}
	contents := map[string][]byte{
		"app.gs.js":                 []byte(`import{a as b,c}from"./app.gs.chunk-shared.js";var d=()=>import("./app.gs.chunk-deferred.js");`),
		"app.gs.chunk-shared.js":    []byte(`export{a,c};`),
		"app.gs.chunk-deferred.js":  []byte(`import{a}from"./app.gs.chunk-shared.js";import "./app.gs.chunk-deferred2.js";`),
		"app.gs.chunk-deferred2.js": []byte(`var e=1;`),
	}
	startup := startupFiles(files, contents)
	assert.Equal(t, map[string]bool{"app.gs.js": true, "app.gs.chunk-shared.js": true}, startup)

	// A bundle of one file is loaded at startup.
	assert.Equal(t, map[string]bool{"app.wasm": true}, startupFiles([]string{"app.wasm"}, map[string][]byte{"app.wasm": {0, 'a', 's', 'm'}}))
}
