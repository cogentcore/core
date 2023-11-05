// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"time"
)

// Web containts the configuration information for building for web and creating
// the HTML page that loads a Go wasm app and its resources.
type Web struct {

	// A placeholder background color for the application page to display before
	// its stylesheets are loaded.
	//
	// DEFAULT: #2d2c2c.
	BackgroundColor string

	// The theme color for the application. This affects how the OS displays the
	// app (e.g., PWA title bar or Android's task switcher).
	//
	// DEFAULT: #2d2c2c.
	ThemeColor string

	// The text displayed while loading a page. Load progress can be inserted by
	// including "{progress}" in the loading label.
	//
	// DEFAULT: "{progress}%".
	LoadingLabel string

	// The page language.
	//
	// DEFAULT: en.
	Lang string

	// The page title.
	Title string

	// The page description.
	Description string

	// The page authors.
	Author string

	// The page keywords.
	Keywords []string

	// The path of the default image that is used by social networks when
	// linking the app.
	Image string

	// The interval between each app auto-update while running in a web browser.
	// Zero or negative values deactivates the auto-update mechanism.
	//
	// Default is 0.
	AutoUpdateInterval time.Duration

	// The environment variables that are passed to the progressive web app.
	//
	// Reserved keys:
	// - GOAPP_VERSION
	// - GOAPP_GOAPP_STATIC_RESOURCES_URL
	Env map[string]string

	// The HTTP header to retrieve the WebAssembly file content length.
	//
	// Content length finding falls back to the Content-Length HTTP header when
	// no content length is found with the defined header.
	WasmContentLengthHeader string

	// The template used to generate app-worker.js. The template follows the
	// text/template package model.
	//
	// By default set to DefaultAppWorkerJS, changing the template have very
	// high chances to mess up go-app usage. Any issue related to a custom app
	// worker template is not supported and will be closed.
	ServiceWorkerTemplate string
}
