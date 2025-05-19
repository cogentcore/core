// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command metricsonly extracts font metrics from font files,
// discarding all the glyph outlines and other data.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/cli"
	"github.com/go-text/typesetting/font/opentype"
)

//go:generate core generate -add-types -add-funcs

func ExtractMetrics(fname, outfile string, debug bool) error {
	if debug {
		fmt.Println(fname, "->", outfile)
	}
	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	font, err := opentype.NewLoader(f)
	if err != nil {
		return err
	}
	// full list from roboto:
	// GSUB OS/2 STAT cmap gasp glyf head hhea hmtx loca maxp name post prep

	// minimal effective list: you have to exclude both loca and glyf -- if
	// you have loca it needs glyf, but otherwise fine to exclude.
	// include := []string{"head", "hhea", "htmx", "maxp", "name", "cmap"}
	include := []string{"head", "hhea", "htmx", "maxp", "name", "cmap"}
	tags := font.Tables()
	tables := make([]opentype.Table, len(tags))
	var taglist []string
	for i, tag := range tags {
		if debug {
			taglist = append(taglist, tag.String())
		}
		skip := true
		for _, in := range include {
			if tag.String() == in {
				skip = false
				break
			}
		}
		if skip {
			continue
		}
		tables[i].Tag = tag
		tables[i].Content, err = font.RawTable(tag)
		if tag.String() == "name" {
			fmt.Println("name:", string(tables[i].Content))
		}
	}
	if debug {
		fmt.Println("\t", taglist)
	}
	content := opentype.WriteTTF(tables)
	return os.WriteFile(outfile, content, 0666)
}

type Config struct {
	// Files to extract metrics from.
	Files []string `flag:"f,files" posarg:"all"`

	// directory to output the metrics only files.
	Output string `flag:"output,o"`

	// emit debug info while processing. todo: use verbose for this!
	Debug bool `flag:"d,debug"`
}

// Extract reads fonts and extracts metrics, saving to given output directory.
func Extract(c *Config) error {
	if c.Output != "" {
		err := os.MkdirAll(c.Output, 0777)
		if err != nil {
			return err
		}
	}
	var errs []error
	for _, fn := range c.Files {
		_, fname := filepath.Split(fn)
		outfile := filepath.Join(c.Output, fname)
		err := ExtractMetrics(fn, outfile, c.Debug)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func main() { //types:skip
	opts := cli.DefaultOptions("metricsonly", "metricsonly extracts font metrics from font files, discarding all the glyph outlines and other data.")
	cli.Run(opts, &Config{}, Extract)
}
