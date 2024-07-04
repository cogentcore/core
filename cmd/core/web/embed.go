// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	_ "embed"
)

var (

	// appWorkerJS is the template used in [makeAppWorkerJS] to generate
	// app-worker.js.
	//
	//go:embed embed/app-worker.js
	appWorkerJS string

	// wasmExecJS is the wasm_exec.js file.
	//
	//go:embed embed/wasm_exec.js
	wasmExecJS string

	// appJS is the string used for [appJSTmpl].
	//
	//go:embed embed/app.js
	appJS string

	// manifestJSON is the string used for [manifestJSONTmpl].
	//
	//go:embed embed/manifest.webmanifest
	manifestJSON string

	// appCSS is the string used for app.css.
	//
	//go:embed embed/app.css
	appCSS string

	// indexHTML is the string used for [indexHTMLTmpl].
	//
	//go:embed embed/index.html
	indexHTML string
)
