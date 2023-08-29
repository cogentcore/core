// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packman

import (
	"fmt"
	"os/exec"
	"strings"
)

// Command is a command that can be used for installing and updating a package
type Command struct {
	Name string
	Args []string
}

// Commands contains a set of commands for each operating system
type Commands map[string][]*Command

// RunCmd runs the given command and returns the combined output
// and any error encountered. If there is an error running the
// command, RunCmd wraps it and includes the output of the command
// in the resulting error message.
func RunCmd(cmd *exec.Cmd) ([]byte, error) {
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error running command: %w; command returned output:\n%s", err, out)
	}
	return out, nil
}

// CmdString returns a string representation of the given command.
func CmdString(cmd *exec.Cmd) string {
	if cmd.Args == nil {
		return "<nil>"
	}
	return strings.Join(cmd.Args, " ")
}

// ArgsString returns a string representation of the given
// command arguments.
func ArgsString(args []string) string {
	if args == nil {
		return "<nil>"
	}
	return strings.Join(args, " ")
}
