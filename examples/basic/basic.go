// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"goki.dev/grease"
)

//go:generate gtigen -add-funcs

type Config struct {

	// the name of the user
	Name string `grease:"name,nm,n" desc:"the name of the user"`

	// the age of the user
	Age int `desc:"the age of the user"`

	// whether the user likes Go
	LikesGo bool `desc:"whether the user likes Go"`

	Build BuildConfig `cmd:"build"`

	Server Server
	Client Client

	// the directory to build in
	Dir string `desc:"the directory to build in"`
}

type BuildConfig struct {

	// the target platform to build for
	Target string `grease:"target,build-target" posarg:"0" desc:"the target platform to build for"`

	// the platform to build the executable for
	Platform string `posarg:"1" required:"-" desc:"the platform to build the executable for"`
}

type Server struct {

	// the server platform
	Platform string `desc:"the server platform"`
}

type Client struct {

	// the client platform
	Platform string `nest:"-" desc:"the client platform"`
}

// Build builds the app for the config platform and target. It builds apps
// across platforms using the GOOS and GOARCH environment variables and a
// suitable C compiler located on the system.
//
// It is the main command used during a local development workflow, and
// it serves as a direct replacement for go build when building GoKi
// apps. In addition to the basic capacities of go build, Build supports
// cross-compiling CGO applications with ease. Also, it handles the
// bundling of icons and fonts into the executable.
//
// Build also uses GoMobile to support the building of .apk and .app
// files for Android and iOS mobile platforms, respectively. Its simple,
// unified, and configurable API for building applications makes it
// the best way to build applications, whether for local debug versions
// or production releases.
func Build(c *Config) error {
	fmt.Println("Building for target", c.Build.Target, "and platform", c.Build.Platform, "- user likes go:", c.LikesGo)
	return nil
}

// Run runs the app for the given user.
func Run(c *Config) error {
	fmt.Println("Running for user", c.Name, "- likes go:", c.LikesGo)
	return nil
}

// Mod configures module information.
func Mod(c *Config) error {
	fmt.Println("running mod")
	return nil
}

// ModTidy tidies module information.
//
//grease:cmd -name "mod tidy"
func ModTidy(c *Config) error {
	fmt.Println("running mod tidy")
	return nil
}

// ModTidyRemote tidies module information for the remote.
//
//grease:cmd -name "mod tidy remote"
func ModTidyRemote(c *Config) error {
	fmt.Println("running mod tidy remote")
	return nil
}

// ModTidyRemoteSetURL tidies module information for the remote
// and sets its URL.
//
//grease:cmd -name "mod tidy remote set-url"
func ModTidyRemoteSetURL(c *Config) error {
	fmt.Println("running mod tidy remote set-url")
	return nil
}

//gti:skip
func main() {
	opts := grease.DefaultOptions("basic", "Basic", "Basic is a basic example application made with Grease.")
	grease.Run(opts, &Config{}, Build, Run, Mod, ModTidy, ModTidyRemote, ModTidyRemoteSetURL)
}
