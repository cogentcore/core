// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/cli"
	"cogentcore.org/core/text/csl"
)

//go:generate core generate -add-types -add-funcs

type Config struct {

	// CSL JSON formatted file with the library of references to lookup citations in.
	Refs string `flag:"r,refs" posarg:"0"`

	// Directory with markdown files to extract citations from.
	// Defaults to current directory if empty.
	Dir string `flag:"d,dir"`

	// File name to write the formatted references to.
	// Defaults to references.md if empty.
	Output string `flag:"o,output"`

	// File name to write the subset of cited reference data to.
	// Defaults to citedrefs.json if empty.
	CitedData string `flag:"c,cited"`

	// heading to add to the top of the references file.
	// Include markdown heading syntax, e.g., ##
	// Defaults to ## References if empty.
	Heading string `flag:"h,heading"`

	// style is the citation style to generate.
	// Defaults to APA if empty.
	Style csl.Styles `flag:"s,style"`
}

// Generate extracts citations and generates resulting references file.
func Generate(c *Config) error {
	refs, err := csl.Open(c.Refs)
	if err != nil {
		return err
	}
	kl := csl.NewKeyList(refs)

	if c.Dir == "" {
		c.Dir = "./"
	}
	mds := fsx.Filenames(c.Dir, ".md")
	if len(mds) == 0 {
		return errors.New("No .md files found in: " + c.Dir)
	}
	if c.Output == "" {
		c.Output = filepath.Join(c.Dir, "references.md")
	}
	of, err := os.Create(c.Output)
	if errors.Log(err) != nil {
		return err
	}
	defer of.Close()

	cited, err := csl.GenerateMarkdown(of, os.DirFS(c.Dir), c.Heading, kl, c.Style, mds...)
	cf := c.CitedData
	if cf == "" {
		cf = "citedrefs.json"
	}
	csl.SaveKeyList(cited, cf)
	return err
}

func main() { //types:skip
	opts := cli.DefaultOptions("mdcite", "mdcites extracts markdown citations from .md files in a directory, and writes a references file with the resulting citations, using the default APA style or the specified one.")
	cli.Run(opts, &Config{}, Generate)
}
