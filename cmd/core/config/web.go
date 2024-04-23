// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"time"
)

// Web containts the configuration information for building for web and creating
// the HTML page that loads a Go wasm app and its resources.
type Web struct { //types:add

	// Port is the port to serve the page at when using the serve command.
	Port string `default:"8080"`

	// RandomVersion is whether to make the app worker version random.
	// It is enabled by default and should be kept on for easy deployment.
	RandomVersion bool `default:"true"`

	// Gzip is whether to gzip the app.wasm file that is built in the build command
	// and serve it as a gzip-encoded file in the run command.
	Gzip bool

	// A placeholder background color for the application page to display before
	// its stylesheets are loaded.
	//
	// DEFAULT: #2d2c2c.
	BackgroundColor string `default:"#2d2c2c"`

	// The theme color for the application. This affects how the OS displays the
	// app (e.g., PWA title bar or Android's task switcher).
	//
	// DEFAULT: #2d2c2c.
	ThemeColor string `default:"#2d2c2c"`

	// The page language.
	//
	// DEFAULT: en.
	Lang string `default:"en"`

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
	// Default is 10 seconds.
	AutoUpdateInterval time.Duration `default:"10s"`

	// If specified, make this page a Go import vanity URL with this
	// module URL, pointing to the GitHub repository specified by GithubVanityURL
	// (eg: cogentcore.org/core).
	VanityURL string

	// If VanityURL is specified, the underlying GitHub repository for the vanity URL
	// (eg: cogentcore/core).
	GithubVanityRepository string

	// The environment variables that are passed to the progressive web app.
	//
	// Reserved keys:
	// - GOAPP_STATIC_RESOURCES_URL
	Env map[string]string

	// The HTTP header to retrieve the WebAssembly file content length.
	//
	// Content length finding falls back to the Content-Length HTTP header when
	// no content length is found with the defined header.
	WasmContentLengthHeader string
}
