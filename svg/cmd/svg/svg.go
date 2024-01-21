// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate -add-types -add-funcs

import (
	"strings"

	"cogentcore.org/core/grease"
	"cogentcore.org/core/svg"
)

func main() { //gti:skip
	opts := grease.DefaultOptions("svg", "svg", "Command line tools for rendering and creating svg files")
	grease.Run(opts, &Config{}, Render)
}

type Config struct {

	// Input is the filename of the input file
	Input string `posarg:"0"`

	// Output is the filename of the output file.
	// Defaults to input with .png instead of .svg.
	Output string `flag:"o,output"`

	Render RenderConfig `cmd:"render"`
}

type RenderConfig struct {

	// Width is the width of the rendered image
	Width int `posarg:"1"`

	// Height is the height of the rendered image.
	// Defaults to width.
	Height int `posarg:"2" required:"-"`
}

// Render renders the svg file to an image.
//
//grease:cmd -root
func Render(c *Config) error {
	if c.Render.Height == 0 {
		c.Render.Height = c.Render.Width
	}
	sv := svg.NewSVG(c.Render.Width, c.Render.Height)
	sv.Norm = true
	err := sv.OpenXML(c.Input)
	if err != nil {
		return err
	}
	sv.Render()
	if c.Output == "" {
		c.Output = strings.TrimSuffix(c.Input, ".svg") + ".png"
	}
	return sv.SavePNG(c.Output)
}
