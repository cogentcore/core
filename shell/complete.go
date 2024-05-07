// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"os"
	"path/filepath"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/parse/complete"
)

// CompleteMatch is the [complete.MatchFunc] for the shell.
func (sh *Shell) CompleteMatch(data any, text string, posLn, posCh int) (md complete.Matches) {
	md.Seed = text
	comps := complete.Completions{}
	entries := errors.Log1(os.ReadDir(sh.Config.Dir))
	for _, entry := range entries {
		icon := icons.File
		if entry.IsDir() {
			icon = icons.Folder
		}
		comps = append(comps, complete.Completion{
			Text: entry.Name(),
			Icon: string(icon),
			Desc: filepath.Join(sh.Config.Dir, entry.Name()),
		})
	}
	md.Matches = complete.MatchSeedCompletion(comps, md.Seed)
	return md
}
