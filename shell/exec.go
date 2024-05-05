// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"os"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/reflectx"
)

// Exec executes the given command string, handling the given arguments appropriately.
// If there is any error, it fatally logs it. It forwards output to [os.Stdout] and
// [os.Stderr] appropriately.
func Exec(cmd any, args ...any) {
	scmd, sargs := execArgs(cmd, args...)
	errors.Log(ExecConfig.Run(scmd, sargs...))
}

// ExecConfig is the [exec.Config] used in [Exec].
var ExecConfig = &exec.Config{
	Env:    map[string]string{},
	Stdout: os.Stdout,
	Stderr: os.Stderr,
	Stdin:  os.Stdin,
}

// Output executes the given command string, handling the given arguments
// appropriately. If there is any error, it fatally logs it. It returns the
// stdout as a string and forwards stderr to [os.Stderr] appropriately.
func Output(cmd any, args ...any) string {
	scmd, sargs := execArgs(cmd, args...)
	out := errors.Log1(OutputConfig.Output(scmd, sargs...))
	return out
}

// OutputConfig is the [exec.Config] used in [Output].
var OutputConfig = &exec.Config{
	Env:    map[string]string{},
	Stderr: os.Stderr,
	Stdin:  os.Stdin,
}

// execArgs converts the given command and arguments into strings.
func execArgs(cmd any, args ...any) (string, []string) {
	scmd := reflectx.ToString(cmd)
	sargs := make([]string, len(args))
	for i, a := range args {
		sargs[i] = reflectx.ToString(a)
	}
	return scmd, sargs
}
