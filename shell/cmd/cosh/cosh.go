// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command cosh is an interactive cli for running and compiling Cogent Shell (cosh).
package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/cli"
	"cogentcore.org/core/shell"
	"cogentcore.org/core/shell/interpreter"
	"github.com/ergochat/readline"
	"github.com/traefik/yaegi/interp"
)

//go:generate core generate -add-types -add-funcs

// Config is the configuration information for the cosh cli.
type Config struct {

	// Input is the name of the input file to run/compile.
	Input string `posarg:"0" required:"-"`

	// Output is the name of the Go file to output to.
	// It defaults to the input file with .cosh changed to .go.
	Output string `cmd:"build" posarg:"1" required:"-"`
}

func main() { //types:skip
	opts := cli.DefaultOptions("cosh", "An interactive tool for running and compiling Cogent Shell (cosh).")
	cli.Run(opts, &Config{}, Run, Build)
}

// Run runs the specified cosh file. If no file is specified,
// it runs an interactive shell that allows the user to input cosh.
func Run(c *Config) error { //cli:cmd -root
	if c.Input == "" {
		return Interactive(c)
	}
	b, err := os.ReadFile(c.Input)
	if err != nil {
		return err
	}
	in := interpreter.NewInterpreter(interp.Options{})
	err = in.Eval(string(b))
	return err
}

// Interactive runs an interactive shell that allows the user to input cosh.
func Interactive(c *Config) error {
	in := interpreter.NewInterpreter(interp.Options{})

	rl, err := readline.NewFromConfig(&readline.Config{
		Listener: func(line []rune, pos int, key rune) (newLine []rune, newPos int, ok bool) {
			if key != '\t' {
				return line, pos, true
			}
			line = slices.Delete(line, pos-1, pos) // get rid of tab
			pos -= 1
			sline := string(line)
			md := in.Shell.CompleteMatch(nil, sline, 0, pos)
			if len(md.Matches) == 0 {
				return line, pos, true
			}
			ed := in.Shell.CompleteEdit(nil, sline, pos, md.Matches[0], md.Seed)
			return []rune(ed.NewText), len(ed.NewText), true
		},
	})
	if err != nil {
		return err
	}
	defer rl.Close()
	log.SetOutput(rl.Stderr()) // redraw the prompt correctly after log output

	for {
		rl.SetPrompt(in.Prompt())
		line, err := rl.ReadLine()
		if errors.Is(err, readline.ErrInterrupt) {
			continue
		}
		if errors.Is(err, io.EOF) {
			os.Exit(0)
		}
		if err != nil {
			return err
		}
		in.Eval(line)
	}
}

// Build builds the specified input cosh file to the specified output Go file.
func Build(c *Config) error {
	if c.Input == "" {
		return fmt.Errorf("need input file")
	}
	if c.Output == "" {
		c.Output = strings.TrimSuffix(c.Input, filepath.Ext(c.Input)) + ".go"
	}
	return shell.NewShell().TranspileFile(c.Input, c.Output)
}
