// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate -add-types -add-funcs

import (
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

	// Output is the filename of the output file
	Output string `flag:"o,output"`

	Render RenderConfig `cmd:"render"`
}

type RenderConfig struct {

	// Width is the width of the rendered image
	Width int

	// Height is the height of the rendered image
	Height int
}

// Render renders the svg file to an image.
//
//grease:cmd -root
func Render(c *Config) error {
	sv := svg.NewSVG(c.Render.Width, c.Render.Height)
	sv.Norm = true
	err := sv.OpenXML(c.Input)
	if err != nil {
		return err
	}
	sv.Render()
	return sv.SavePNG(c.Output)
}
