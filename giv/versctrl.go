// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

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
