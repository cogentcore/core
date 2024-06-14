// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"slices"
	"strings"

	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/exec"
)

// RunSubShell runs given command as a new separate Shell, if the command
// cmd has a .cosh suffix and the file exists at given path or in current directory.
// The args are pasted as an additional line of code at the end of the .cosh file,
// to control what happens in running the file.  Also looks for a file of the same name
// as the given command with a .cosh suffix in the current directory.
func (sh *Shell) RunSubShell(cmdIO *exec.CmdIO, errOk, output bool, cmd string, args ...string) (bool, string) {
	out := ""
	if strings.HasSuffix(cmd, ".cosh") {
		if !errors.Log1(dirs.FileExists(cmd)) {
			return false, out
		}
	} else {
		var ok bool
		cmd, ok = sh.CurDirScript(cmd)
		if !ok {
			return false, out
		}
	}
	expr := ""
	if len(args) > 0 {
		expr = strings.Join(args, " ")
	}
	aargs := []any{cmd}
	if expr != "" {
		aargs = append(aargs, "-e", expr)
	}
	out = sh.Exec(errOk, false, output, "cosh", aargs...)
	return true, out
}

// CurDirScript returns name of script file with .cosh suffix appended,
// in current directory.  returns false if not found.  Files are cached
// for faster repeated access.
func (sh *Shell) CurDirScript(cmd string) (string, bool) {
	cmd += ".cosh"
	if sh.cwdScriptFilesDir != sh.Config.Dir {
		sh.cwdScriptFiles = dirs.ExtFilenames("./", ".cosh")
		sh.cwdScriptFilesDir = sh.Config.Dir
	}
	if i := slices.Index(sh.cwdScriptFiles, cmd); i >= 0 {
		return cmd, true
	}
	return cmd, false
}
