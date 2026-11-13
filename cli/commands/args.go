// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/sharedfile"
)

// Args are the arguments last used for the subcommands of the commands,
// by command, in a structure parallel to that of the [Cmd] and [SubCmd],
// which are saved in a JSON file (see [OpenArgs]). They are set from the
// form of a subcommand when its command line is run, and set in the form
// when it is shown again.
type Args struct {

	// Cmds are the arguments of the commands, by command: "git".
	Cmds map[string]*CmdArgs

	// Scripts are the values of the arguments of the scripts, by the key
	// of the script (see [scriptKey]), and by the name of the argument:
	// "true" for a switch that is on.
	Scripts map[string]map[string]string

	// file is the file the arguments are saved in, which other processes
	// (apps) share.
	file sharedfile.File[argsData]

	// mu protects the arguments and the saving of the file.
	mu sync.Mutex
}

// CmdArgs are the arguments last used for the subcommands of a [Cmd].
type CmdArgs struct {

	// SubCmds are the arguments of the subcommands, by the name of the
	// subcommand: "commit", or [MainName].
	SubCmds map[string]*SubCmdArgs
}

// SubCmdArgs are the arguments last used for a [SubCmd]: only those that
// were used.
type SubCmdArgs struct {

	// Args are the other arguments, such as files, as typed in a shell.
	Args string `json:",omitempty"`

	// Flags are the values of the flags that were used, by the name of the
	// flag ("--message"): "true" for a flag without a value.
	Flags map[string]string `json:",omitempty"`
}

// OpenArgs returns the arguments saved in the given JSON file, which are
// empty if it does not exist yet, and are saved in it when they change.
// The file may be shared by other processes (apps): it is read again when
// another process changes it.
func OpenArgs(filename string) *Args {
	as := &Args{file: sharedfile.File[argsData]{Path: filename}, Cmds: map[string]*CmdArgs{}}
	as.mu.Lock()
	as.syncLocked()
	as.mu.Unlock()
	return as
}

// argsData are the data of the file of the [Args].
type argsData struct {
	Cmds    map[string]*CmdArgs
	Scripts map[string]map[string]string `json:",omitempty"`
}

// syncLocked reads the file of the arguments if another process has
// changed it, under the mutex.
func (as *Args) syncLocked() {
	if !as.file.Changed() {
		return
	}
	errors.Log(as.file.Reload())
	if as.file.Data.Cmds != nil {
		as.Cmds = as.file.Data.Cmds
	}
	as.Scripts = as.file.Data.Scripts
}

// get returns the arguments last used for the given subcommand, or nil.
func (as *Args) get(sc *SubCmd) *SubCmdArgs {
	root, _ := splitCmd(sc.Cmd)
	as.mu.Lock()
	defer as.mu.Unlock()
	as.syncLocked()
	if ca := as.Cmds[root]; ca != nil {
		return ca.SubCmds[sc.Name]
	}
	return nil
}

// Set sets the arguments last used for the given subcommand from the given
// values of its form (see [SubCmd.FormStruct]), and saves them.
func (as *Args) Set(sc *SubCmd, values any) error {
	sa := sc.ArgsOf(values)
	root, _ := splitCmd(sc.Cmd)
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.file.Update(func(d *argsData) {
		if d.Cmds != nil {
			as.Cmds = d.Cmds
		}
		ca := as.Cmds[root]
		if ca == nil {
			ca = &CmdArgs{}
			as.Cmds[root] = ca
		}
		if ca.SubCmds == nil {
			ca.SubCmds = map[string]*SubCmdArgs{}
		}
		ca.SubCmds[sc.Name] = sa
		d.Cmds = as.Cmds
		as.Scripts = d.Scripts
	})
}

// SetValues sets the given values of the form of the given subcommand
// (see [SubCmd.FormStruct]) to the arguments last used for it, if any.
func (as *Args) SetValues(sc *SubCmd, values any) {
	as.get(sc).SetValues(sc, values)
}

// scriptKey returns the key of the given script of the given command (""
// for one of an entry) in the [Args.Scripts].
func scriptKey(root string, s *Script) string {
	return root + "/" + s.Name
}

// ScriptValues returns the values of the arguments last used for the given
// script of the given command ("" for one of an entry), by name, which are
// their defaults for those not used yet.
func (as *Args) ScriptValues(root string, s *Script) map[string]string {
	values := map[string]string{}
	var last map[string]string
	if as != nil {
		as.mu.Lock()
		as.syncLocked()
		last = as.Scripts[scriptKey(root, s)]
		as.mu.Unlock()
	}
	for _, a := range s.Args {
		v, ok := last[a.Name]
		if !ok {
			v = a.Default
		}
		values[a.Name] = v
	}
	return values
}

// SetScriptValues sets the values of the arguments last used for the given
// script of the given command ("" for one of an entry) to the given ones,
// by name, and saves them.
func (as *Args) SetScriptValues(root string, s *Script, values map[string]string) error {
	if as == nil {
		return nil
	}
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.file.Update(func(d *argsData) {
		if d.Cmds != nil {
			as.Cmds = d.Cmds
		}
		as.Scripts = d.Scripts
		if as.Scripts == nil {
			as.Scripts = map[string]map[string]string{}
		}
		as.Scripts[scriptKey(root, s)] = values
		d.Cmds = as.Cmds
		d.Scripts = as.Scripts
	})
}
