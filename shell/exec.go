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
		fmt.Fprintln(sh.Config.StdIO.Err, err.Error())
	} else {
		sh.AddError(err)
	}
	return err
}

// ExecArgs processes the args to given exec command,
// handling all of the input / outpu redirection and
// file globbing, homedir expansion, etc.
func (sh *Shell) ExecArgs(errok bool, cmd any, args ...any) (string, []string) {
	scmd := reflectx.ToString(cmd)
	sargs := make([]string, len(args))
	for i, a := range args {
		s := reflectx.ToString(a)
		s, err := homedir.Expand(s)
		sh.HandleArgErr(errok, err)
		// todo: filepath.Glob
		sargs[i] = s
	}
	if len(sargs) == 0 {
		return scmd, sargs
	}
	if scmd == "@" {
		if sargs[0] == "0" {
			sh.SSHActive = "" // local
		} else {
			if _, ok := sh.SSHClients[sargs[0]]; ok {
				// todo: this needs to be in a stack, popped after command is run
				sh.SSHActive = sargs[0]
			} else {
				sh.HandleArgErr(errok, fmt.Errorf("cosh: ssh connection named: %q not found", sargs[0]))
			}
		}
		if len(sargs) > 1 {
			scmd = sargs[1]
			sargs = sargs[2:]
		} else {
			return "", nil
		}
	}
	for i := 0; i < len(sargs); i++ {
		s := sargs[i]
		switch {
		case len(s) > 0 && s[0] == '>':
			sargs = sh.OutToFile(errok, sargs, i)
		}
	}
	return scmd, sargs
}

// OutToFile processes the > arg that sends output to a file
func (sh *Shell) OutToFile(errok bool, sargs []string, i int) []string {
	n := len(sargs)
	s := sargs[i]
	sn := len(s)
	fn := ""
	narg := 1
	if i < n-1 {
		fn = sargs[1+1]
		narg = 2
	}
	appn := false
	errf := false
	switch {
	case sn > 1 && s[1] == '>':
		appn = true
		if sn > 2 && s[2] == '&' {
			errf = true
		}
	case sn > 1 && s[1] == '&':
		errf = true
	case sn > 1:
		fn = s[1:]
		narg = 1
	}
	fmt.Println(s, appn, errf, narg, fn)
	sargs = slices.Delete(sargs, i, i+narg)
	if fn == "" {
		sh.HandleArgErr(errok, fmt.Errorf("cosh: no output file specified"))
		return sargs
	}
	var f *os.File
	var err error
	if appn {
		f, err = os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	} else {
		f, err = os.Create(fn)
	}
	if err == nil {
		sh.Config.StdIO.PushOut(f)
		if errf {
			sh.Config.StdIO.PushErr(f)
		}
	} else {
		sh.HandleArgErr(errok, err)
	}
	return sargs
}

// CancelExecution calls the Cancel() function if set.
func (sh *Shell) CancelExecution() {
	if sh.Cancel != nil {
		sh.Cancel()
	}
}
