// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xe

import (
	"io"
	"os"

	"log/slog"

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
	// exit code of 1 when there is an error running a command. It should
	// only be used by end-user apps, not libraries, which should use more
	// robust and idiomatic error handling.
	Fatal bool
	// PrintOnly is whether to only print commands that would be run and
	// not actually run them. It can be used, for example, for safely testing
	// an app.
	PrintOnly bool

	// The directory to execute commands in. If it is unset,
	// commands are run in the current directory.
	Dir string
	// Env contains any additional environment variables specified.
	// The current environment variables will also be passed to the
	// command, but they will be overridden by any variables here
	// if there are conflicts.
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

// Major returns the default [Config] object for a major command,
// based on [grog.UserLevel]. It should be used for commands that
// are central to an app's logic and are more important for the user
// to know about and be able to see the output of. It results in
// commands and output being printed with a [grog.UserLevel] of
// [slog.LevelInfo] or below, whereas [Minor] results in that when
// it is [slog.LevelDebug] or below. Most commands in a typical use
// case should be Major, which is why the global helper functions
// operate on it.
func Major() *Config {
	if grog.UserLevel <= slog.LevelInfo {
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
	return &Config{
		Buffer: true,
		Env:    map[string]string{},
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
		Errors: os.Stderr,
	}
}

// Minor returns the default [Config] object for a minor command,
// based on [grog.UserLevel]. It should be used for commands that
// support an app behind the scenes and are less important for the
// user to know about and be able to see the output of. It results in
// commands and output being printed with a [grog.UserLevel] of
// [slog.LevelDebug] or below, whereas [Major] results in that when
// it is [slog.LevelInfo] or below.
func Minor() *Config {
	if grog.UserLevel <= slog.LevelDebug {
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
	return &Config{
		Buffer: true,
		Env:    map[string]string{},
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
		Errors: os.Stderr,
	}
}

// GetWriter returns the appropriate writer to use based on the given writer and error.
// If the given error is non-nil, the returned writer is guaranteed to be non-nil,
// with [Config.Stderr] used as a backup. Otherwise, the returned writer will only
// be non-nil if the passed one is.
func (c *Config) GetWriter(w io.Writer, err error) io.Writer {
	res := w
	if res == nil && err != nil {
		res = c.Stderr
	}
	return res
}

// PrintCmd uses [GetWriter] to print the given command to [Config.Commands]
// or [Config.Stderr], based on the given error and the config settings.
// A newline is automatically inserted.
func (c *Config) PrintCmd(cmd string, err error) {
	cmds := c.GetWriter(c.Commands, err)
	if cmds != nil {
		if c.Dir != "" {
			cmds.Write([]byte(grog.SuccessColor(c.Dir) + ": "))
		}
		cmds.Write([]byte(grog.CmdColor(cmd) + "\n"))
	}
}

func (c *Config) SetBuffer(buffer bool) *Config {
	c.Buffer = buffer
	return c
}

func (c *Config) SetFatal(fatal bool) *Config {
	c.Fatal = fatal
	return c
}

func (c *Config) SetPrintOnly(printOnly bool) *Config {
	c.PrintOnly = printOnly
	return c
}

func (c *Config) SetDir(dir string) *Config {
	c.Dir = dir
	return c
}

func (c *Config) SetEnv(key, val string) *Config {
	c.Env[key] = val
	return c
}
