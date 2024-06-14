// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/parse/complete"
	"github.com/mitchellh/go-homedir"
)

// CompleteMatch is the [complete.MatchFunc] for the shell.
func (sh *Shell) CompleteMatch(data any, text string, posLine, posChar int) (md complete.Matches) {
	comps := complete.Completions{}
	text = text[:posChar]
	md.Seed = complete.SeedPath(text)
	fullPath := complete.SeedSpace(text)
	fullPath = errors.Log1(homedir.Expand(fullPath))
	parent := strings.TrimSuffix(fullPath, md.Seed)
	dir := filepath.Join(sh.Config.Dir, parent)
	if filepath.IsAbs(parent) {
		dir = parent
	}
	entries := errors.Log1(os.ReadDir(dir))
	for _, entry := range entries {
		icon := icons.File
		if entry.IsDir() {
			icon = icons.Folder
		}
		name := strings.ReplaceAll(entry.Name(), " ", `\ `) // escape spaces
		comps = append(comps, complete.Completion{
			Text: name,
			Icon: icon,
			Desc: filepath.Join(sh.Config.Dir, name),
		})
	}
	if parent == "" {
		for cmd := range sh.Builtins {
			comps = append(comps, complete.Completion{
				Text: cmd,
				Icon: icons.Terminal,
				Desc: "Builtin command: " + cmd,
			})
		}
		for cmd := range sh.Commands {
			comps = append(comps, complete.Completion{
				Text: cmd,
				Icon: icons.Terminal,
				Desc: "Command: " + cmd,
			})
		}
		// todo: write something that looks up all files on path -- should cache that per
		// path string setting
	}
	md.Matches = complete.MatchSeedCompletion(comps, md.Seed)
	return md
}

// CompleteEdit is the [complete.EditFunc] for the shell.
func (sh *Shell) CompleteEdit(data any, text string, cursorPos int, completion complete.Completion, seed string) (ed complete.Edit) {
	return complete.EditWord(text, cursorPos, completion.Text, seed)
}

// ReadlineCompleter implements [github.com/ergochat/readline.AutoCompleter].
type ReadlineCompleter struct {
	Shell *Shell
}

func (rc *ReadlineCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	text := string(line)
	md := rc.Shell.CompleteMatch(nil, text, 0, pos)
	res := [][]rune{}
	for _, match := range md.Matches {
		after := strings.TrimPrefix(match.Text, md.Seed)
		if md.Seed != "" && after == match.Text {
			continue // no overlap
		}
		if match.Icon == icons.Folder {
			after += string(filepath.Separator)
		} else {
			after += " "
		}
		res = append(res, []rune(after))
	}
	return res, len(md.Seed)
}
