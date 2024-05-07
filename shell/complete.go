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
)

// CompleteMatch is the [complete.MatchFunc] for the shell.
func (sh *Shell) CompleteMatch(data any, text string, posLn, posCh int) (md complete.Matches) {
	comps := complete.Completions{}
	text = text[:posCh]
	md.Seed = complete.SeedPath(text)
	fullPath := complete.SeedSpace(text)
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
		comps = append(comps, complete.Completion{
			Text: entry.Name(),
			Icon: icon,
			Desc: filepath.Join(sh.Config.Dir, entry.Name()),
		})
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
