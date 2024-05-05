// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"

	"cogentcore.org/core/cli"
	"cogentcore.org/core/shell/interpreter"
	"github.com/ergochat/readline"
)

//go:generate core generate -add-types -add-funcs

type Config struct {
	File string `flag:"file,f"`
}

// Run compiles and runs the file
func Run(c *Config) error {
	if c.File == "" {
		return fmt.Errorf("File not specified")
	}
	b, err := os.ReadFile(c.File)
	if err != nil {
		return err
	}
	in := interpreter.NewInterpreter()
	err = in.Eval(string(b))
	return err
}

// Interactive runs an interactive shell
func Interactive(c *Config) error {
	// see readline.NewFromConfig for advanced options:
	rl, err := readline.New("> ")
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()
	log.SetOutput(rl.Stderr()) // redraw the prompt correctly after log output
	in := interpreter.NewInterpreter()
	for {
		rl.SetPrompt(in.Prompt())
		line, err := rl.ReadLine()
		// `err` is either nil, io.EOF, readline.ErrInterrupt, or an unexpected
		// condition in stdin:
		if err != nil {
			return err
		}
		// `line` is returned without the terminating \n or CRLF:
		// fmt.Fprintf(rl, "you wrote: %s\n", line)
		in.Eval(line)
	}
	return nil
}

func main() { //types:skip
	opts := cli.DefaultOptions("cosh", "The Cogent Core Shell.")
	cli.Run(opts, &Config{}, Run, Interactive)
}
