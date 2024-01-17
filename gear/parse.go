// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gear

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"cogentcore.org/core/grr"
	"cogentcore.org/core/xe"
	"github.com/iancoleman/strcase"
)

// Parse uses the help messages of the app to fill in its data fields.
func (cm *Cmd) Parse() error {
	blocks, err := cm.GetBlocks()
	if err != nil {
		return err
	}
	return cm.SetFromBlocks(blocks)
}

// SetFromBlocks sets the information of the command from the given [ParseBlock] objects.
func (cm *Cmd) SetFromBlocks(blocks []ParseBlock) error {
	cmdsDone := map[string]bool{}
	for _, block := range blocks {
		// a - indicates a flag
		if strings.HasPrefix(block.Name, "-") {
			flag := &Flag{}
			flag.Doc = block.Doc
			fields := strings.Fields(block.Name)
			for _, field := range fields {
				if strings.HasPrefix(field, "-") {
					name := strings.Trim(field, ",")
					hasNonDashCharcters := strings.ContainsFunc(name, func(r rune) bool {
						return r != '-'
					})
					// if we have no non-dash characters, we aren't actually a flag
					if !hasNonDashCharcters {
						continue
					}
					flag.Names = append(flag.Names, name)
					continue
				}
				flag.Type = field
			}
			if len(flag.Names) == 0 {
				continue
			}
			slices.SortFunc(flag.Names, func(a, b string) int {
				return cmp.Compare(len(a), len(b))
			})
			flag.Name = flag.Names[len(flag.Names)-1]
			cm.Flags = append(cm.Flags, flag)
			continue
		}
		// if we have no -, we are probably a command

		// however, if we have something other than lowercase letters, -, and _,
		// we probably aren't a command, so we probably got included in here by mistake
		hasUnallowedRunes := strings.ContainsFunc(block.Name, func(r rune) bool {
			return !(unicode.IsLower(r) || r == '-' || r == '_')
		})
		if hasUnallowedRunes {
			continue
		}

		// if the normalized version of our old command contains the normalized
		// version of our block name, it is probably just referencing
		// the old command somewhere (not specifying a new command that is the exact same),
		// which leads to infinite recursion, so we just skip it.
		kbnm := strcase.ToKebab(block.Name)
		if strings.Contains(strcase.ToKebab(cm.Cmd), kbnm) {
			continue
		}
		cmdsContains := slices.ContainsFunc(cm.Cmds, func(c *Cmd) bool {
			return strings.Contains(strcase.ToKebab(c.Cmd), kbnm)
		})
		if cmdsContains {
			continue
		}

		cmd := NewCmd(cm.Cmd + " " + block.Name)

		if len(strings.Fields(cmd.Cmd)) > 2 {
			continue
		}

		if cmdsDone[cmd.Cmd] {
			continue
		}
		cmdsDone[cmd.Cmd] = true

		cmd.Doc = block.Doc

		cm.Cmds = append(cm.Cmds, cmd)

		// there is no helpful information to extract from help commands
		if kbnm != "help" {
			// Now we must recursively parse the subcommand and any of its subcommands.
			// Errors here are not fatal, as various subcommands could be mistakes, so
			// we just log them and move on.
			grr.Log(cmd.Parse())
		}
	}
	return nil
}

// ParseBlock is a block of parsed content containing the name of something and
// the documentation for it.
type ParseBlock struct {
	Name string
	Doc  string
}

// GetBlocks gets the [ParseBlock] objects for this command.
func (cm *Cmd) GetBlocks() ([]ParseBlock, error) {
	h, err := cm.GetHelp()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(h, "\n")

	blocks := []ParseBlock{}
	prevNspc := 0
	curBlock := ParseBlock{}
	for _, line := range lines {
		// get the effective number of spaces at the start of this line
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

		tsline := strings.TrimSpace(line) // trim space line

		// If we have more than one space at the start, we check for more than
		// one space in the middle (see the comments below for more information)
		if nspc > 1 {
			rtsline := []rune(tsline) // rune trim space line
			mlnspc := 0               // middle line num spaces
			gotMiddleSpace := false
			for i, r := range rtsline {
				if r == '\t' {
					mlnspc += 4
				} else if unicode.IsSpace(r) {
					mlnspc += 1
				}
				// if we have already had spaces and now have a non-space,
				// then we have broken the space sequence and do not have a middle space.
				if mlnspc > 0 && !unicode.IsSpace(r) {
					break
				}
				// If we have more than one effective space in the middle of the line, we
				// interpret that as a separator between the name and doc of a standalone block.
				// Therefore, we make a block with this info, push it onto the stack, clear any
				// previous info, and then continue to the next line.
				if mlnspc > 1 {
					before := strings.TrimSpace(string(rtsline[:i]))
					after := strings.TrimSpace(string(rtsline[i:]))
					block := ParseBlock{Name: before, Doc: after}
					blocks = append(blocks, block)
					curBlock = ParseBlock{}
					prevNspc = mlnspc
					gotMiddleSpace = true
					break
				}
			}
			if gotMiddleSpace {
				continue
			}
		}

		// If we have more than one space and previously had nothing,
		// we are the start of a new block
		if nspc > 1 && prevNspc == 0 {
			curBlock.Name = tsline
			prevNspc = nspc
		} else if nspc > 1 && nspc >= prevNspc {
			// If we are at the same or higher level relative to the start
			// of this block, we are part of its documentation

			// we add a space to separate lines
			if curBlock.Doc != "" {
				curBlock.Doc += " "
			}
			curBlock.Doc += tsline
			prevNspc = nspc
		} else if nspc < prevNspc {
			// If we have moved backward from a block, we are done with it
			// and push it onto the stack of blocks. We do not add the block
			// if it empty.
			if curBlock.Name != "" {
				blocks = append(blocks, curBlock)
			}
			if nspc > 1 {
				prevNspc = nspc
				curBlock = ParseBlock{Name: tsline}
			} else {
				prevNspc = 0
				curBlock = ParseBlock{}
			}
		}
	}
	return blocks, nil
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
