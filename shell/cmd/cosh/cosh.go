// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command cosh is an interactive cli for running and compiling Cogent Shell (cosh).
package main

import (
	"log"
	"os"

	"cogentcore.org/core/cli"
	"cogentcore.org/core/shell/interpreter"
	"github.com/ergochat/readline"
)

//go:generate core generate -add-types -add-funcs

// Config is the configuration information for the cosh cli.
type Config struct {

	// File is the file to run/compile.
	File string `posarg:"0" required:"-"`
}

// Run runs the specified cosh file. If no file is specified,
// it runs an interactive shell that allows the user to input cosh.
func Run(c *Config) error { //cli:cmd -root
	if c.File == "" {
		return Interactive(c)
	}
	b, err := os.ReadFile(c.File)
	if err != nil {
		return err
	}
	in := interpreter.NewInterpreter()
	err = in.Eval(string(b))
	return err
}

// Interactive runs an interactive shell that allows the user to input cosh.
func Interactive(c *Config) error {
	rl, err := readline.New("> ")
	if err != nil {
		return err
	}
	defer rl.Close()
	log.SetOutput(rl.Stderr()) // redraw the prompt correctly after log output
	in := interpreter.NewInterpreter()
	for {
		rl.SetPrompt(in.Prompt())
		line, err := rl.ReadLine()
		if err != nil {
			return err
		}
		in.Eval(line)
	}
}

func main() { //types:skip
	opts := cli.DefaultOptions("cosh", "An interactive tool for running and compiling Cogent Shell (cosh).")
	cli.Run(opts, &Config{}, Run)
}
