// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log/slog"

	"cogentcore.org/core/cli"
	"cogentcore.org/core/text/spell"
)

//go:generate core generate -add-types -add-funcs

// Config is the configuration information for the dict cli.
type Config struct {

	// InputA is the first input dictionary file
	InputA string `posarg:"0" required:"+"`

	// InputB is the second input dictionary file
	InputB string `posarg:"1" required:"+"`

	// Output is the output file for merge command
	Output string `cmd:"merge" posarg:"2" required:"-"`
}

func main() { //types:skip
	opts := cli.DefaultOptions("dict", "runs dictionary commands")
	cli.Run(opts, &Config{}, Compare, Merge)
}

// Compare compares two dictionaries
func Compare(c *Config) error { //cli:cmd -root
	a, err := spell.OpenDict(c.InputA)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	b, err := spell.OpenDict(c.InputB)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	fmt.Printf("In %s not in %s:\n", c.InputA, c.InputB)
	for aw := range a {
		if !b.Exists(aw) {
			fmt.Println(aw)
		}
	}
	fmt.Printf("\n########################\nIn %s not in %s:\n", c.InputB, c.InputA)
	for bw := range b {
		if !a.Exists(bw) {
			fmt.Println(bw)
		}
	}
	return nil
}

// Merge combines two dictionaries
func Merge(c *Config) error { //cli:cmd
	return nil
}
