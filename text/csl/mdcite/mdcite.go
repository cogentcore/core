// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/cli"
	"cogentcore.org/core/text/csl"
)

//go:generate core generate -add-types -add-funcs

type Config struct {

	// CSL JSON formatted file with the library of references to lookup citations in.
	Refs string `flag:"r,refs" required:"+"`

	// Directory with markdown files to extract citations from.
	// Defaults to current directory if empty.
	Dir string `flag:"d,dir" required:"-"`

	// File name to write the formatted references to.
	// Defaults to references.md if empty.
	Output string `flag:"o,output" required:"-"`

	// File name to write the subset of cited reference data to.
	// Defaults to citedrefs.json if empty.
	CitedData string `flag:"c,cited" required:"-"`

	// heading to add to the top of the references file.
	// Include markdown heading syntax, e.g., ##
	// Defaults to ## References if empty.
	Heading string `flag:"h,heading" required:"-"`

	// style is the citation style to generate.
	// Defaults to APA if empty.
	Style csl.Styles `flag:"s,style"  required:"-"`
}

// Generate extracts citations and generates resulting references file.
func Generate(c *Config) error {
	refs, err := csl.Open(c.Refs)
	if err != nil {
		return err
	}
	kl := csl.NewKeyList(refs)
	cited, err := csl.GenerateMarkdown(c.Dir, c.Output, c.Heading, kl, c.Style)
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
