// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate -add-types -add-funcs

import (
	"path/filepath"
	"strings"

	"cogentcore.org/core/cli"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/svg"
)

func main() { //gti:skip
	opts := cli.DefaultOptions("svg", "svg", "Command line tools for rendering and creating svg files")
	cli.Run(opts, &Config{}, Render, EmbedImage)
}

type Config struct {

	// Input is the filename of the input file
	Input string `posarg:"0"`

	// Output is the filename of the output file.
	// Defaults to input with the extension changed to the output format.
	Output string `flag:"o,output"`

	// Fill, if specified, indicates to fill the background of
	// the svg with the specified color in CSS format.
	Fill string

	Render RenderConfig `cmd:"render"`
}

type RenderConfig struct {

	// Width is the width of the rendered image
	Width int `posarg:"1"`

	// Height is the height of the rendered image.
	// Defaults to width.
	Height int `posarg:"2" required:"-"`
}

// Render renders the input svg file to the output image file.
//
//grease:cmd -root
func Render(c *Config) error {
	if c.Render.Height == 0 {
		c.Render.Height = c.Render.Width
	}
	sv := svg.NewSVG(c.Render.Width, c.Render.Height)
	err := ApplyFill(c, sv)
	if err != nil {
		return err
	}
	err = sv.OpenXML(c.Input)
	if err != nil {
		return err
	}
	sv.Render()
	if c.Output == "" {
		c.Output = strings.TrimSuffix(c.Input, filepath.Ext(c.Input)) + ".png"
	}
	return sv.SavePNG(c.Output)
}

// EmbedImage embeds the input image file into the output svg file.
func EmbedImage(c *Config) error {
	sv := svg.NewSVG(0, 0)
	err := ApplyFill(c, sv)
	if err != nil {
		return err
	}
	img := svg.NewImage(&sv.Root)
	err = img.OpenImage(c.Input, 0, 0)
	if err != nil {
		return err
	}
	sv.Root.ViewBox.Size.SetPoint(img.Pixels.Bounds().Size())
	if c.Output == "" {
		c.Output = strings.TrimSuffix(c.Input, filepath.Ext(c.Input)) + ".svg"
	}
	return sv.SaveXML(c.Output)
}

// ApplyFill applies [Config.Fill] to the given [svg.SVG].
func ApplyFill(c *Config, sv *svg.SVG) error { //gti:skip
	if c.Fill == "" {
		return nil
	}
	bg, err := gradient.FromString(c.Fill)
	if err != nil {
		return err
	}
	sv.Background = bg
	return nil
}
