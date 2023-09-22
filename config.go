// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xe

import (
	"io"
	"os"

	"github.com/fatih/color"
)

// Config contains the configuration information that
// controls the behavior of xe. It is passed to most
// high-level functions, and a default version of it
// can be easily constructed using [DefaultConfig].
type Config struct {
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
	// Fatal is whether to fatally exit programs with [os.Exit] and an
	// exit code of 1 when there is an error running a program.
	Fatal bool
	// Env contains any additional environment variables specified.
	Env map[string]string
	// CmdColor is the color formatting function used on commands.
	CmdColor func(format string, a ...any) string
	// ErrColor is the color formatting function used on errors.
	ErrColor func(format string, a ...any) string
}

// VerboseConfig returns a default verbose [Config] object.
// It prints everything to [os.Stdout] or [os.Stderr].
func VerboseConfig() *Config {
	return &Config{
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Stdin:    os.Stdin,
		Commands: os.Stdout,
		Errors:   os.Stderr,
		Fatal:    false,
		Env:      map[string]string{},
		CmdColor: color.New(color.FgCyan, color.Bold).Sprintf,
		ErrColor: color.New(color.FgRed).Sprintf,
	}
}

// ErrorConfig returns a default error [Config] object.
// It prints errors to [os.Stderr] and prints nothing else.
func ErrorConfig() *Config {
	return &Config{
		Stdout:   nil,
		Stderr:   os.Stderr,
		Stdin:    os.Stdin,
		Commands: nil,
		Errors:   os.Stderr,
		Fatal:    false,
		Env:      map[string]string{},
		CmdColor: color.New(color.FgCyan, color.Bold).Sprintf,
		ErrColor: color.New(color.FgRed).Sprintf,
	}
}

// SilentConfig returns a default silent [Config] object.
// it prints nothing, not even errors.
func SilentConfig() *Config {
	return &Config{
		Stdout:   nil,
		Stderr:   nil,
		Stdin:    os.Stdin,
		Commands: nil,
		Errors:   nil,
		Fatal:    false,
		Env:      map[string]string{},
		CmdColor: color.New(color.FgCyan, color.Bold).Sprintf,
		ErrColor: color.New(color.FgRed).Sprintf,
	}
}

// FatalConfig returns a default fatal [Config] object.
// It prints everything to [os.Stdout] or [os.Stderr],
// and fatally exits on any error.
func FatalConfig() *Config {
	return &Config{
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Stdin:    os.Stdin,
		Commands: os.Stdout,
		Errors:   os.Stderr,
		Fatal:    true,
		Env:      map[string]string{},
		CmdColor: color.New(color.FgCyan, color.Bold).Sprintf,
		ErrColor: color.New(color.FgRed).Sprintf,
	}
}
