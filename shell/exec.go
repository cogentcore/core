// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"log"
	"os"

	"cogentcore.org/core/base/exec"
	"github.com/mattn/go-shellwords"
)

// Exec executes the given command string, parsing and separating any arguments.
// If there is any error, it fatally logs it. It returns the stdout of the command,
// in addition to forwarding output to [os.Stdout] and [os.Stderr] appropriately.
func Exec(cmd string) string {
	args, err := shellwords.Parse(cmd)
	if err != nil {
		log.Fatalln("shell.Exec: error parsing arguments:", err)
	}
	if len(args) == 0 {
		log.Fatalln("shell.Exec: no arguments")
	}

	ecmd := args[0]
	var eargs []string
	if len(args) > 1 {
		eargs = args[1:]
	}
	out, _ := ExecConfig.Output(ecmd, eargs...)
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
