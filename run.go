// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted in part from: https://github.com/magefile/mage
// Copyright presumably by Nate Finch, primary contributor
// Apache License, Version 2.0, January 2004

package xe

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-shellwords"
	"goki.dev/grog"
)

// Args returns a string parsed into separate args
// that can be passed into run commands.
func Args(cfg *Config, str string) []string {
	args, err := shellwords.Parse(str)
	if err != nil {
		if cfg.Errors != nil {
			cfg.Errors.Write([]byte(err.Error())) // note: we want to use the results inline, so no error return
		}
		if cfg.Fatal {
			os.Exit(1)
		}
	}
	return args
}

// RunCmd returns a function that will call Run with the given command. This is
// useful for creating command aliases to make your scripts easier to read, like
// this:
//
//	 // in a helper file somewhere
//	 var g0 = sh.RunCmd("go")  // go is a keyword :(
//
//	 // somewhere in your main code
//		if err := g0("install", "github.com/gohugo/hugo"); err != nil {
//			return err
//	 }
//
// Args passed to command get baked in as args to the command when you run it.
// Any args passed in when you run the returned function will be appended to the
// original args.  For example, this is equivalent to the above:
//
//	var goInstall = sh.RunCmd("go", "install") goInstall("github.com/gohugo/hugo")
//
// RunCmd uses Exec underneath, so see those docs for more details.
func (c *Config) RunCmd(cmd string, args ...string) func(args ...string) error {
	return func(args2 ...string) error {
		return c.Run(cmd, append(args, args2...)...)
	}
}

// OutCmd is like RunCmd except the command returns the output of the
// command.
func (c *Config) OutCmd(cmd string, args ...string) func(args ...string) (string, error) {
	return func(args2 ...string) (string, error) {
		return c.Output(cmd, append(args, args2...)...)
	}
}

// RunSh runs given full command string with args formatted
// as in a standard shell command
func (c *Config) RunSh(cstr string) error {
	args, err := shellwords.Parse(cstr)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		err := fmt.Errorf("command %q was not parsed correctly into content", cstr)
		if c.Errors != nil {
			c.Errors.Write([]byte(grog.ErrorColor(err.Error())))
		}
		if c.Fatal {
			os.Exit(1)
		}
		return err
	}
	cmd := args[0]
	var rmdr []string
	if len(args) > 1 {
		rmdr = args[1:]
	}
	return c.Run(cmd, rmdr...)
}

// Run runs the given command using the given configuration information and arguments.
func (c *Config) Run(cmd string, args ...string) error {
	_, err := c.Exec(cmd, args...)
	return err
}

// Output runs the command and returns the text from stdout.
func (c *Config) Output(cmd string, args ...string) (string, error) {
	oldStdout := c.Stdout
	// need to use buf to capture output
	buf := &bytes.Buffer{}
	c.Stdout = buf
	_, err := c.Exec(cmd, args...)
	c.Stdout = oldStdout
	if c.Stdout != nil {
		c.Stdout.Write(buf.Bytes())
	}
	return strings.TrimSuffix(buf.String(), "\n"), err
}

// RunSh calls [Config.RunSh] on [Major]
func RunSh(cstr string) error {
	return Major().RunSh(cstr)
}

// Run calls [Config.Run] on [Major]
func Run(cmd string, args ...string) error {
	return Major().Run(cmd, args...)
}

// Output calls [Config.Output] on [Major]
func Output(cmd string, args ...string) (string, error) {
	return Major().Output(cmd, args...)
}
