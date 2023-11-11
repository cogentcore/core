// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gear

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"goki.dev/xe"
)

// flagRegexp matches flags.
// The second submatch contains the flag name(s) with any dashes, commas, and spaces still included.
var flagRegexp = regexp.MustCompile(
	`(?m)` + // multi line
		`(?:\s{1,16}|\t)` + // starting space
		`((?:[ |[,(]+\-[\w-]+)+)`) // flag(s)

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
	h, err := cm.GetHelp()
	if err != nil {
		return err
	}

	// flags := flagRegexp.FindAllStringSubmatch(h, -1)
	// for _, flag := range flags {
	// 	names := flag[1]
	// 	fields := strings.Fields(names)

	// 	f := &Flag{}
	// 	for _, field := range fields {
	// 		name := strings.Trim(field, "-,[(| \t")
	// 		if name != "" {
	// 			f.Names = append(f.Names, name)
	// 		}
	// 	}
	// 	if len(f.Names) == 0 {
	// 		continue
	// 	}
	// 	slices.SortFunc(f.Names, func(a, b string) int {
	// 		return cmp.Compare(len(a), len(b))
	// 	})
	// 	f.Name = f.Names[len(f.Names)-1]
	// 	cm.Flags = append(cm.Flags, f)
	// }

	// cmds := cmdRegexp.FindAllStringSubmatch(h, -1)
	// for _, cmd := range cmds {
	// 	c := NewCmd(cm.Cmd + " " + cmd[1])
	// 	// remove first part of command for name (the app name)
	// 	c.Name = sentencecase.Of(strings.Join(strings.Fields(c.Name)[1:], " "))
	// 	if len(cmd) >= 3 {
	// 		c.Doc = cmd[2]
	// 	}

	// 	cm.Cmds = append(cm.Cmds, c)

	// 	// we don't want to parse the help info for help commands
	// 	if c.Name != "Help" {
	// 		err := c.Parse()
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}
	// }

	lines := strings.Split(h, "\n")

	blocks := []block{}
	prevNspc := 0
	prevBlock := block{}
	for _, line := range lines {
		nspc := 0
		for _, r := range line {
			if r == '\t' {
				nspc += 4
			} else if unicode.IsSpace(r) {
				nspc += 1
			} else {
				break
			}
		}
		if nspc > 1 && prevNspc == 0 {
			prevNspc = nspc
			prevBlock.name = strings.TrimSpace(line)
		} else if nspc >= prevNspc {
			prevBlock.doc += strings.TrimSpace(line)
		} else {
			blocks = append(blocks, prevBlock)
			if nspc > 1 {
				prevNspc = nspc
				prevBlock = block{name: strings.TrimSpace(line)}
			} else {
				prevNspc = 0
				prevBlock = block{}
			}
		}
	}
	for _, block := range blocks {
		fmt.Println("BLOCK", block.name, ":", block.doc)
	}
	return nil
}

type block struct {
	name string
	doc  string
}

// GetHelp gets the help information for the command. It tries various different
// commands and flags to get the help information and only returns an error if all
// of them fail.
func (cm *Cmd) GetHelp() (string, error) {
	fields := strings.Fields(cm.Cmd)
	root := fields[0]

	// try man first
	out, err := xe.Silent().Output("man", fields...)
	if err == nil {
		return out, nil
	}

	hcmds := []string{"help", "--help", "-h"}
	for _, hcmd := range hcmds {
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
