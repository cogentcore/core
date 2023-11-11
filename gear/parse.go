// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gear

import (
	"fmt"
	"regexp"

	"goki.dev/xe"
)

// flagRegexp matches flags.
// The second match is the name of the flag.
var flagRegexp = regexp.MustCompile(`\W\-+([\w\-]+)`)

// type parsing for flagRegexp:
// \W\-+([\w\-]+)([= ]<(\w+)>)?

// cmdRegexp matches commands.
// The second match is the name of the command.
// The fourth match, if it exists, is the description of the command.
var cmdRegexp = regexp.MustCompile(`\n\s{2,}(\w+)(\s{2,}([^\n]+))?`)

// Parse uses the help messages of the app to fill in its data fields.
func (a *App) Parse() error {
	rh, err := a.GetHelp("")
	if err != nil {
		return err
	}

	flags := flagRegexp.FindAllStringSubmatch(rh, -1)
	for _, flag := range flags {
		// second item has the submatch
		a.Flags = append(a.Flags, flag[1])
	}

	cmds := cmdRegexp.FindAllStringSubmatch(rh, -1)
	for _, cmd := range cmds {
		c := NewApp(cmd[1])
		if len(cmd) >= 4 {
			c.Doc = cmd[3]
		}
		fmt.Println(c.Command)
		a.Commands = append(a.Commands, c)
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

// GetHelp gets the help information for the given command, or the root command
// if the command is unspecified. It tries various different commands and flags
// to get the help information and only returns an error if all of them fail.
func (a *App) GetHelp(cmd string) (string, error) {
	hcmds := []string{"help", "--help", "-h"}
	for _, hcmd := range hcmds {
		args := []string{hcmd}
		if cmd != "" {
			args = append(args, cmd)
		}
		out, err := xe.Silent().Output(a.Command, args...)
		if err == nil {
			return out, nil
		}
		if cmd != "" {
			// try both orders
			out, err = xe.Silent().Output(a.Command, cmd, hcmd)
			if err == nil {
				return out, nil
			}
		}
	}
	return "", fmt.Errorf("unable to get help information for command %q of app %q (%q)", cmd, a.Name, a.Command+" "+cmd)
}
