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

	// Gzip is whether to gzip the files of the app bundle that the build
	// command makes (app.wasm for wasm, and app.gs.js and its chunks for js)
	// in place, without changing their names, so that they must be served
	// with the Content-Encoding: gzip header, as the run command does.
	Gzip bool

	// GenerateHTML is whether to generate an HTML version of app content for
	// preview and SEO purposes.
	GenerateHTML bool `default:"true"`

	// HTMLOnly is whether to build a freestanding website consisting only of
	// static HTML pages generated from the app content, without the Go wasm app.
	// It is only supported for apps that use the content package. The pages are
	// the same as the preview pages that are otherwise shown while the app loads.
	HTMLOnly bool

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

	// Styles is a list of URLs to CSS stylesheets that are loaded in the HTML page.
	// This is often used for custom fonts; see https://cogentcore.org/core/font.
	Styles []string

	// FontText is text whose characters the app starts loading the faces
	// of its regular text font for at startup, in addition to the Latin
	// ones; for example, "āΩ" loads the Latin Extended and Greek faces
	// (about 190 KB) for a site with names or math that use them. Without
	// it, those faces load when the first text that needs them is rendered,
	// and the app then styles and lays out everything again.
	FontText string

	// DeferredFunctions are exported functions (package/path.Function) that
	// js builds load when they are first called (GoScript --deferred-function):
	// their packages, and the packages that only they use, are in separate
	// files of the bundle, downloaded and initialized at that first call. Code
	// that runs at startup may refer to them only as function values (and
	// calls), such as cogentcore.org/core/mvmcore.BindTextEditor, the
	// interpreter of edited content code.
	DeferredFunctions []string

	// The interval between each app auto-update while running in a web browser.
	// Zero or negative values deactivate the auto-update mechanism. The auto-update
	// mechanism updates the WASM file for the app from the server if it has changed.
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
