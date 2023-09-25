// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xe

import (
	"io"
	"os"

	"goki.dev/grog"
)

// Config contains the configuration information that
// controls the behavior of xe. It is passed to most
// high-level functions, and a default version of it
// can be easily constructed using [DefaultConfig].
type Config struct {
	// Buffer is whether to buffer the output of Stdout and Stderr,
	// which is necessary for the correct printing of commands and output
	// when there is an error with a command, and for correct coloring
	// on Windows. Therefore, it should be kept at the default value of
	// true in most cases, except for when a command will run for a log
	// time and print output throughout (eg: a log command).
	Buffer bool
	// Fatal is whether to fatally exit programs with [os.Exit] and an
	// exit code of 1 when there is an error running a program.
	Fatal bool
	// The directory to execute commands in. If it is unset,
	// commands are run in the current directory.
	Dir string
	// Env contains any additional environment variables specified.
	Env map[string]string

	// Stdout is the writer to write the standard output of called commands to.
	// It can be set to nil to disable the writing of the standard output.
	Stdout io.Writer
	// Stderr is the writer to write the standard error of called commands to.
	// It can be set to nil to disable the writing of the standard error.
	Stderr io.Writer
	// Stdin is the reader to use as the standard input.
	Stdin io.Reader
	// Commands is the writer to write the string representation of the called commands to.
	// It can be set to nil to disable the writing of the string representations of the called commands.
	Commands io.Writer
	// Errors is the writer to write program errors to.
	// It can be set to nil to disable the writing of program errors.
	Errors io.Writer
}

// Main returns the default [Config] object for a main command,
// based on [grog.UserLevel].
func Main() *Config {
	if grog.UserLevel > grog.Info {
		return &Config{
			Buffer: true,
			Env:    map[string]string{},
			Stderr: os.Stderr,
			Stdin:  os.Stdin,
			Errors: os.Stderr,
		}
	}
	return &Config{
		Buffer:   true,
		Env:      map[string]string{},
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Stdin:    os.Stdin,
		Commands: os.Stdout,
		Errors:   os.Stderr,
	}
}

// Minor returns the default [Config] object for a minor command,
// based on [grog.UserLevel].
func Minor() *Config {
	if grog.UserLevel > grog.Debug {
		return &Config{
			Buffer: true,
			Env:    map[string]string{},
			Stderr: os.Stderr,
			Stdin:  os.Stdin,
			Errors: os.Stderr,
		}
	}
	return &Config{
		Buffer:   true,
		Env:      map[string]string{},
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Stdin:    os.Stdin,
		Commands: os.Stdout,
		Errors:   os.Stderr,
	}
}
