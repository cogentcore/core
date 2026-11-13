// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package commands provides a GUI for calling command line tools, such as
// git, whose many options are hard to remember. The help output of each
// command is parsed into a [Cmd], with its subcommands, such as git commit,
// each a [SubCmd] with its flags, which are cached (see [Commands]). The
// user organizes the commands into categories, edited in a
// [core.TreeTable] (see [Commands.Edit]), which are shown in a command bar
// like a bookmarks bar, or in a menu, with menus of the subcommands of the
// commands (see [Commands.Bar] and [Commands.Menu]). Clicking on a
// subcommand shows a form of its flags and arguments, which builds the
// command line to run, in a terminal for example (see
// [Commands.RunDialog]), starting with the arguments last used for it (see
// [Args]). A [Macro] is a version of a subcommand with values for its
// arguments, some of which are prompted for when it runs, such as the
// [StandardMacros] of git, go, and core.
//
// The [Default] commands are shared by the apps, such as Cogent Files and
// Cogent Code, in the Cogent Core data directory: their files are read
// again when another app changes them.
package commands

import (
	"strings"

	"cogentcore.org/core/base/strcase"
)

// Cmd is a command, such as git, with its subcommands, such as
// git commit, parsed from its help output (see [ParseCmd]).
type Cmd struct {

	// Cmd is the command: "git".
	Cmd string

	// Doc is the documentation of the command, if found.
	Doc string `json:",omitempty"`

	// SubCmds are the subcommands of the command, which are parsed when
	// they are used (see [SubCmd.IsParsed]). A command without
	// subcommands has one, named [MainName], for the command itself.
	SubCmds []*SubCmd

	// Macros are the macros of the subcommands, made by the user, which
	// are at the top of the menu of the command (see [Macro]). They are
	// kept when the command is parsed again.
	Macros []*Macro `json:",omitempty"`

	// Scripts are the scripts of the command, made by the user, which are
	// in the menu of the command after its macros (see [Script]). They are
	// kept when the command is parsed again.
	Scripts []*Script `json:",omitempty"`
}

// IsParsed returns whether the help of the command has been parsed: it
// has subcommands (it may only have macros otherwise, from a file of
// another version).
func (cm *Cmd) IsParsed() bool {
	return len(cm.SubCmds) > 0
}

// MainName is the name of the [SubCmd] of a [Cmd] without subcommands,
// which is the command itself.
const MainName = "Main"

// SubCmd returns the subcommand of the given name, such as "commit" or
// "stash list", or nil if there is none.
func (cm *Cmd) SubCmd(name string) *SubCmd {
	for _, sc := range cm.SubCmds {
		if sc.Name == name {
			return sc
		}
	}
	return nil
}

// HasSubCmds returns whether the command has subcommands, other than its
// [MainName] one.
func (cm *Cmd) HasSubCmds() bool {
	return len(cm.SubCmds) > 1 || (len(cm.SubCmds) == 1 && cm.SubCmds[0].Name != MainName)
}

// SubCmd is a subcommand of a [Cmd], such as git commit, or the command
// itself (see [MainName]), with the information parsed from its help
// output (see [Parse]).
type SubCmd struct {

	// Name is the name of the subcommand, its words after the command:
	// "commit", "stash list", or [MainName] for the command itself.
	Name string

	// Cmd is the command line of the subcommand: "git commit".
	Cmd string

	// Doc is the documentation of the subcommand, if found.
	Doc string `json:",omitempty"`

	// Usage is the usage synopsis of the subcommand, if found.
	Usage string `json:",omitempty"`

	// Flags are the flags of the subcommand.
	Flags []*Flag `json:",omitempty"`

	// Group is the group of the subcommand in the help of its command,
	// such as "Main Porcelain Commands" for git: the heading of its
	// section, which is shown as a submenu, other than the first group.
	Group string `json:",omitempty"`

	// Cmds are the subcommands of the subcommand, such as git stash list,
	// with their documentation (or usage), which are parsed as subcommands
	// of the command when they are used.
	Cmds []*SubCmd `json:",omitempty"`

	// Source is the help command whose output was parsed: "git commit -h",
	// which is empty if the subcommand has not been parsed yet.
	Source string `json:",omitempty"`
}

// IsParsed returns whether the help of the subcommand has been parsed.
func (sc *SubCmd) IsParsed() bool {
	return sc.Source != ""
}

// Label returns the text shown for the subcommand: [MainName], or its
// last word in sentence case: "Commit", "List" for "stash list".
func (sc *SubCmd) Label() string {
	if sc.Name == MainName {
		return MainName
	}
	return label(sc.Cmd)
}

// label returns the last word of the given command in sentence case.
func label(cmd string) string {
	if f := strings.Fields(cmd); len(f) > 0 {
		return strcase.ToSentence(f[len(f)-1])
	}
	return ""
}

// splitCmd splits the given command line into the command and the name
// of its subcommand: "git", "stash list".
func splitCmd(cmd string) (root, name string) {
	f := strings.Fields(cmd)
	if len(f) == 0 {
		return "", ""
	}
	return f[0], strings.Join(f[1:], " ")
}

// Flag is a flag of a [Cmd].
type Flag struct {

	// Name is the canonical (longest) name of the flag, with its dashes:
	// "--message".
	Name string

	// Names are all of the names of the flag, with their dashes, shortest
	// first: "-m", "--message".
	Names []string

	// Value is the hint for the value of the flag, such as "<message>",
	// or empty for a flag without a value, which is on or off.
	Value string `json:",omitempty"`

	// Optional is whether the value is optional, as in "--name[=<value>]":
	// it is attached to the flag, with = for long names.
	Optional bool `json:",omitempty"`

	// Equals is whether the value is attached with =, as in "--name=<value>".
	Equals bool `json:",omitempty"`

	// Negatable is whether the flag has a negated form, as in "--[no-]name".
	Negatable bool `json:",omitempty"`

	// Doc is the documentation of the flag.
	Doc string `json:",omitempty"`
}

// IsBool returns whether the flag has no value: it is on or off.
func (fl *Flag) IsBool() bool {
	return fl.Value == ""
}

// Label returns the label of the flag: its names and value hint.
func (fl *Flag) Label() string {
	s := strings.Join(fl.Names, ", ")
	if fl.Value != "" {
		s += " " + fl.Value
	}
	return s
}

// Arg returns the command line argument(s) for the flag with the given
// value, which is quoted for a shell as needed.
func (fl *Flag) Arg(value string) string {
	if fl.IsBool() {
		return fl.Name
	}
	q := shellQuote(value)
	long := strings.HasPrefix(fl.Name, "--")
	switch {
	case fl.Equals || (fl.Optional && long):
		return fl.Name + "=" + q
	case fl.Optional:
		return fl.Name + q
	}
	return fl.Name + " " + q
}

// shellQuote returns the given string quoted for a POSIX shell, if it has
// any characters that need it.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_./:=,@%+", r))
	}) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
