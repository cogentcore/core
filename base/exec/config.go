// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package exec provides an easy way to execute commands,
// improving the ease-of-use and error handling of the
// standard library os/exec package. For example:
//
//	err := exec.Run("git", "commit", "-am")
//	// or
//	err := exec.RunSh("git commit -am")
//	// or
//	err := exec.Verbose().Run("git", "commit", "-am")
package exec

//go:generate core generate

import (
	"io"
	"os"

	"log/slog"

	"cogentcore.org/core/base/logx"
)

// Config contains the configuration information that
// controls the behavior of exec. It is passed to most
// high-level functions, and a default version of it
// can be easily constructed using [DefaultConfig].
type Config struct { //types:add -setters

	// Buffer is whether to buffer the output of Stdout and Stderr,
	// which is necessary for the correct printing of commands and output
	// when there is an error with a command, and for correct coloring
	// on Windows. Therefore, it should be kept at the default value of
	// true in most cases, except for when a command will run for a log
	// time and print output throughout (eg: a log command).
	Buffer bool

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
	Env map[string]string `set:"-"`

	// Echo is the writer for echoing the command string to.
	// It can be set to nil to disable echoing.
	Echo io.Writer

	// Standard Input / Output management
	StdIO StdIO
}

// major is the config object for [Major] specified through [SetMajor]
var major *Config

// Major returns the default [Config] object for a major command,
// based on [logx.UserLevel]. It should be used for commands that
// are central to an app's logic and are more important for the user
// to know about and be able to see the output of. It results in
// commands and output being printed with a [logx.UserLevel] of
// [slog.LevelInfo] or below, whereas [Minor] results in that when
// it is [slog.LevelDebug] or below. Most commands in a typical use
// case should be Major, which is why the global helper functions
// operate on it. The object returned by Major is guaranteed to be
// unique, so it can be modified directly.
func Major() *Config {
	if major != nil {
		// need to make a new copy so people can't modify the underlying
		res := *major
		return &res
	}
	if logx.UserLevel <= slog.LevelInfo {
		c := &Config{
			Buffer: true,
			Env:    map[string]string{},
			Echo:   os.Stdout,
		}
		c.StdIO.StdAll()
		return c
	}
	c := &Config{
		Buffer: true,
		Env:    map[string]string{},
	}
	c.StdIO.StdAll()
	c.StdIO.Out = nil
	return c
}

// SetMajor sets the config object that [Major] returns. It should
// be used sparingly, and only in cases where there is a clear property
// that should be set for all commands. If the given config object is
// nil, [Major] will go back to returning its default value.
func SetMajor(c *Config) {
	major = c
}

// minor is the config object for [Minor] specified through [SetMinor]
var minor *Config

// Minor returns the default [Config] object for a minor command,
// based on [logx.UserLevel]. It should be used for commands that
// support an app behind the scenes and are less important for the
// user to know about and be able to see the output of. It results in
// commands and output being printed with a [logx.UserLevel] of
// [slog.LevelDebug] or below, whereas [Major] results in that when
// it is [slog.LevelInfo] or below. The object returned by Minor is
// guaranteed to be unique, so it can be modified directly.
func Minor() *Config {
	if minor != nil {
		// need to make a new copy so people can't modify the underlying
		res := *minor
		return &res
	}
	if logx.UserLevel <= slog.LevelDebug {
		c := &Config{
			Buffer: true,
			Env:    map[string]string{},
			Echo:   os.Stdout,
		}
		c.StdIO.StdAll()
		return c
	}
	c := &Config{
		Buffer: true,
		Env:    map[string]string{},
	}
	c.StdIO.StdAll()
	c.StdIO.Out = nil
	return c
}

// SetMinor sets the config object that [Minor] returns. It should
// be used sparingly, and only in cases where there is a clear property
// that should be set for all commands. If the given config object is
// nil, [Minor] will go back to returning its default value.
func SetMinor(c *Config) {
	minor = c
}

// verbose is the config object for [Verbose] specified through [SetVerbose]
var verbose *Config

// Verbose returns the default [Config] object for a verbose command,
// based on [logx.UserLevel]. It should be used for commands
// whose output are central to an application; for example, for a
// logger or app runner. It results in commands and output being
// printed with a [logx.UserLevel] of [slog.LevelWarn] or below,
// whereas [Major] and [Minor] result in that when it is [slog.LevelInfo]
// and [slog.levelDebug] or below, respectively. The object returned by
// Verbose is guaranteed to be unique, so it can be modified directly.
func Verbose() *Config {
	if verbose != nil {
		// need to make a new copy so people can't modify the underlying
		res := *verbose
		return &res
	}
	if logx.UserLevel <= slog.LevelWarn {
		c := &Config{
			Buffer: true,
			Env:    map[string]string{},
			Echo:   os.Stdout,
		}
		c.StdIO.StdAll()
		return c
	}
	c := &Config{
		Buffer: true,
		Env:    map[string]string{},
	}
	c.StdIO.StdAll()
	c.StdIO.Out = nil
	return c
}

// SetVerbose sets the config object that [Verbose] returns. It should
// be used sparingly, and only in cases where there is a clear property
// that should be set for all commands. If the given config object is
// nil, [Verbose] will go back to returning its default value.
func SetVerbose(c *Config) {
	verbose = c
}

// silent is the config object for [Silent] specified through [SetSilent]
var silent *Config

// Silent returns the default [Config] object for a silent command,
// based on [logx.UserLevel]. It should be used for commands that
// whose output/input is private and needs to be always hidden from
// the user; for example, for a command that involves passwords.
// It results in commands and output never being printed. The object
// returned by Silent is guaranteed to be unique, so it can be modified directly.
func Silent() *Config {
	if silent != nil {
		// need to make a new copy so people can't modify the underlying
		res := *silent
		return &res
	}
	c := &Config{
		Buffer: true,
		Env:    map[string]string{},
	}
	c.StdIO.In = os.Stdin
	return c
}

// SetSilent sets the config object that [Silent] returns. It should
// be used sparingly, and only in cases where there is a clear property
// that should be set for all commands. If the given config object is
// nil, [Silent] will go back to returning its default value.
func SetSilent(c *Config) {
	silent = c
}

// GetWriter returns the appropriate writer to use based on the given writer and error.
// If the given error is non-nil, the returned writer is guaranteed to be non-nil,
// with [Config.Stderr] used as a backup. Otherwise, the returned writer will only
// be non-nil if the passed one is.
func (c *Config) GetWriter(w io.Writer, err error) io.Writer {
	res := w
	if res == nil && err != nil {
		res = c.StdIO.Err
	}
	return res
}

// PrintCmd uses [GetWriter] to print the given command to [Config.Echo]
// or [Config.Stderr], based on the given error and the config settings.
// A newline is automatically inserted.
func (c *Config) PrintCmd(cmd string, err error) {
	cmds := c.GetWriter(c.Echo, err)
	if cmds != nil {
		if c.Dir != "" {
			cmds.Write([]byte(logx.SuccessColor(c.Dir) + ": "))
		}
		cmds.Write([]byte(logx.CmdColor(cmd) + "\n"))
	}
}

// PrintCmd calls [Config.PrintCmd] on [Major]
func PrintCmd(cmd string, err error) {
	Major().PrintCmd(cmd, err)
}

// SetEnv sets the given environment variable.
func (c *Config) SetEnv(key, val string) *Config {
	c.Env[key] = val
	return c
}
