// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"os/exec"
)

// ChangeRec is version control change-log record
type ChangeRec struct {
	Date     FileTime `desc:"date/time when change made"`
	CommitID string   `desc:"unique identifier for the commit (git SHA, svn rev)"`
	Author   string   `desc:"author name"`
	Email    string   `desc:"author email"`
	Message  string   `desc:"commit message summarizing changes"`
	Files    string   `desc:"files changed in this commit"`
}

// ChangeLog is a record of changes committed from within the Gide system to
// the version control system for this project.  Use the Log command for your
// VCS to see all changes.
type ChangeLog []ChangeRec

// Add adds a record to the change log, at the top of the log
func (cl *ChangeLog) Add(cr ChangeRec) {
	sz := len(*cl)
	*cl = append(*cl, cr)
	if sz > 0 {
		copy((*cl)[1:], (*cl)[0:sz])
	}
	(*cl)[0] = cr
}

// VersCtrlSystems is a list of supported Version Control Systems -- use these
// names in commands to select commands for the current VCS for this project
// (i.e., use shortest version of name, typically three letters)
var VersCtrlSystems = []string{"Git", "SVN"}

// VersCtrlName is the name of a version control system
type VersCtrlName string

// VersCtrlFiles is a map of signature files that indicate which VC is in use
var VersCtrlFiles = map[string]string{
	"Git": ".git",
	"SVN": ".svn",
}

// GitCmds is a map of actions and the Git command string
var GitCmds = map[string]string{
	"Add":            "add",
	"Delete":         "rm",
	"DeleteRepoOnly": "rm --cached",
	"Rename":         "mv",
}

// SvnCmds is a map of actions and the Subversion command string
var SvnCmds = map[string]string{
	"Add":    "add",
	"Delete": "rm",
	"Rename": "mv",
}

// ExecGitCmd calls os.exec to execute a Git command
func ExecGitCmd(action, arg2, arg3 string) {
	arg1 := GitCmds[action]
	oscmd := exec.Command("git", arg1, arg2, arg3)
	stdoutStderr, err := oscmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", err)
	}
	fmt.Printf("%s\n", stdoutStderr)
}

// ExecSvnCmd calls os.exec to execute a Svn command
func ExecSvnCmd(action, arg2, arg3 string) {

}
