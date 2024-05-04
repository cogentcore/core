// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"os"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/reflectx"
)

// Exec executes the given command string, parsing and separating any arguments.
// If there is any error, it fatally logs it. It returns the stdout of the command,
// in addition to forwarding output to [os.Stdout] and [os.Stderr] appropriately.
func Exec(cmd any, args ...any) string {
	scmd := reflectx.ToString(cmd)
	sargs := make([]string, len(args))
	for i, a := range args {
		sargs[i] = reflectx.ToString(a)
	}
	out, _ := ExecConfig.Output(scmd, sargs...)
	return out
}

// ExecConfig is the [exec.Config] used in [Exec].
var ExecConfig = &exec.Config{
	Fatal:  true,
	Env:    map[string]string{},
	Stdout: os.Stdout,
	Stderr: os.Stderr,
	Stdin:  os.Stdin,
}
