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

	// the target platform to build for
	BuildTarget string `grease:"build-target,target" cmd:"build" posarg:"0" desc:"the target platform to build for"`

	Server Server
	Client Client

	Directory string `grease:"dir" nest:"-"`
	Dir       string `nest:"-"`
}

type Server struct {

	// the server platform
	Platform string `desc:"the server platform"`
}

type Client struct {

	// the client platform
	Platform string `nest:"-" desc:"the client platform"`
}

// Build builds the app for the given platform.
func Build(c *Config) error {
	fmt.Println("Building for platform", c.BuildTarget, "- likes go:", c.LikesGo)
	return nil
}

// Run runs the app for the given user.
//
//grease:cmd -root
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

func main() {
	opts := grease.DefaultOptions("basic", "Basic", "Basic is a basic example application made with Grease.")
	grease.Run(opts, &Config{}, Build, Run, Mod, ModTidy, ModTidyRemote, ModTidyRemoteSetURL)
}
