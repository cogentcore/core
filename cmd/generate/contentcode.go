// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generate

import (
	"fmt"
	"slices"

	"cogentcore.org/core/cmd/config"
	"cogentcore.org/core/content/contentcode"
)

// ContentCode compiles the Go and Goal code blocks of the content of the app
// in dir into the app if it is a content app (see [contentcode.App]), with
// the options of cc (from core.toml). It does nothing if they are up to date.
func ContentCode(dir string, cc *config.ContentCode) error {
	c, err := contentcode.App(dir)
	if err != nil || c == nil {
		return err
	}
	for _, im := range cc.Imports {
		if !slices.Contains(c.Imports, im) {
			c.Imports = append(c.Imports, im)
		}
	}
	c.GoalRecord = c.GoalRecord && cc.GoalRecord
	if _, err := contentcode.Generate(c); err != nil {
		return fmt.Errorf("compiling the code blocks of the content: %w", err)
	}
	return nil
}
