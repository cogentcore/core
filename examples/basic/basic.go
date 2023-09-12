// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"

	"goki.dev/grease"
)

type Config struct {

	// the name of the user
	Name string `desc:"the name of the user"`

	// the age of the user
	Age int `desc:"the age of the user"`

	// whether the user likes Go
	LikesGo bool `desc:"whether the user likes Go"`

	// the target platform to build for
	BuildTarget string `desc:"the target platform to build for"`
}

// Build builds the app for the given platform.
//
//gti:add
func Build(c *Config) error {
	if c.BuildTarget == "" {
		return errors.New("missing build target")
	}
	fmt.Println("Building for platform", c.BuildTarget)
	return nil
}

func main() {
	grease.AppName = "basic"
	grease.AppTitle = "Basic"
	grease.AppAbout = "Basic is a basic example application made with Grease."
	grease.DefaultFiles = []string{"config.toml"}
	err := grease.Run(&Config{}, Build)
	if err != nil {
		fmt.Println(err)
	}
}
