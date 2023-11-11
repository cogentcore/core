// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gear

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"goki.dev/glop/sentencecase"
	"goki.dev/xe"
)

// flagRegexp matches flags.
// The second submatch contains the flag name(s) with any dashes, commas, and spaces still included.
var flagRegexp = regexp.MustCompile(
	`(?m)` + // multi line
		`^(?:\s{2,16}|\t)` + // starting space
		`((?:\W+\-[\w-]+)+)`) // flag(s)

// type parsing for flagRegexp:
// \W\-+([\w\-]+)([= ]<(\w+)>)?

// cmdRegexp matches commands.
// The second submatch is the name of the command.
// The third submatch, if it exists, is the description of the command.
var cmdRegexp = regexp.MustCompile(
	`(?m)` + // multi line
		`^(?:\s{2,16}|\t)` + // starting space
		`(\w[\w\-\.]*)` + // command
		`\s{2,}` + // space between command and doc
		`([^\n]*)`) // doc

// Parse uses the help messages of the app to fill in its data fields.
func (cm *Cmd) Parse() error {
	rh, err := cm.GetHelp()
	if err != nil {
		return err
	}

	flags := flagRegexp.FindAllStringSubmatch(rh, -1)
	for _, flag := range flags {
		names := flag[1]
		fields := strings.Fields(names)

		f := &Flag{}
		f.Names = make([]string, len(fields))
		for i, field := range fields {
			f.Names[i] = strings.Trim(field, "-, \t")
		}
		slices.Sort(f.Names)
		f.Name = f.Names[len(f.Names)-1]
		cm.Flags = append(cm.Flags, f)
	}

	cmds := cmdRegexp.FindAllStringSubmatch(rh, -1)
	for _, cmd := range cmds {
		c := NewCmd(cm.Cmd + " " + cmd[1])
		// remove first part of command for name (the app name)
		c.Name = sentencecase.Of(strings.Join(strings.Fields(c.Name)[1:], " "))
		if len(cmd) >= 3 {
			c.Doc = cmd[2]
		}
		cm.Cmds = append(cm.Cmds, c)
	}

	// lines := strings.Split(rh, "\n")
	// for _, line := range lines {
	// 	for i, r := range line {
	// 		ispc := unicode.IsSpace(r)
	// 		if i == 0 && !ispc {
	// 			break
	// 		}
	// 		if !ispc {

	// 		}

	// 	}
	// }
	return nil
}

// GetHelp gets the help information for the command. It tries various different
// commands and flags to get the help information and only returns an error if all
// of them fail.
func (cm *Cmd) GetHelp() (string, error) {
	hcmds := []string{"help", "--help", "-h"}
	for _, hcmd := range hcmds {
		fields := strings.Fields(cm.Cmd)
		root := fields[0]
		args := append([]string{hcmd}, fields[1:]...)
		out, err := xe.Silent().Output(root, args...)
		if err == nil {
			return out, nil
		}
		if len(fields) > 1 {
			// try both orders
			args := append(fields[1:], hcmd)
			out, err = xe.Silent().Output(root, args...)
			if err == nil {
				return out, nil
			}
		}
	}
	return "", fmt.Errorf("unable to get help information for %q (command %q)", cm.Name, cm.Cmd)
}
