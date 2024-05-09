// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"fmt"
	"os"
	"slices"

	"cogentcore.org/core/base/reflectx"
	"github.com/mitchellh/go-homedir"
)

func (sh *Shell) HandleArgErr(errok bool, err error) error {
	if err == nil {
		return err
	}
	if errok {
		fmt.Fprintln(sh.Config.Stderr, err.Error())
	} else {
		sh.AddError(err)
	}
	return err
}

// ExecArgs processes the args to given exec command,
// handling all of the input / outpu redirection and
// file globbing, homedir expansion, etc.
func (sh *Shell) ExecArgs(errok bool, cmd any, args ...any) (string, []string, []*os.File) {
	scmd := reflectx.ToString(cmd)
	sargs := make([]string, len(args))
	var files []*os.File
	for i, a := range args {
		s := reflectx.ToString(a)
		s, err := homedir.Expand(s)
		sh.HandleArgErr(errok, err)
		// todo: filepath.Glob
		sargs[i] = s
	}
	for i := 0; i < len(sargs); i++ {
		n := len(sargs)
		s := sargs[i]
		switch {
		case s == ">": // todo: handle >>
			if i < n-1 { // todo: should only be at the end!
				fn := sargs[i+1]
				sargs = slices.Delete(sargs, i, i+2)
				o, err := os.Create(fn)
				if err == nil {
					files = append(files, o)
					sh.Config.PushStdout(o)
				} else {
					sh.HandleArgErr(errok, err)
				}
			} else {
				sh.HandleArgErr(errok, fmt.Errorf("cosh: no output file specified"))
			}
		}
	}
	return scmd, sargs, files
}

// CloseFiles closes given files in list
func (sh *Shell) CloseFiles(files []*os.File) {
	for _, f := range files {
		if f != nil {
			f.Close()
			sh.Config.PopStdout() // todo: need to know what kind
		}
	}
}

// CancelExecution calls the Cancel() function if set.
func (sh *Shell) CancelExecution() {
	if sh.Cancel != nil {
		sh.Cancel()
	}
}
