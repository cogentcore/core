// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"fmt"
	"os"
)

func (sh *Shell) Cd(args ...string) error {
	if len(args) != 1 {
		return sh.AddError(fmt.Errorf("cd requires one argument: the directory"))
	}
	dir := args[0]
	// sh.Config.Dir = args[0]
	os.Chdir(dir)
	return nil
}

// example alias:
// shell list(args ...string) {
// 	ls -la {args...}
// }
