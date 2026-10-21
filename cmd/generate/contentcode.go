// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generate

import (
	"fmt"

	"cogentcore.org/core/content/contentcode"
)

// ContentCode compiles the Go code blocks of the content of the app in dir
// into the app if it is a content app (see [contentcode.App]). It does
// nothing if they are up to date.
func ContentCode(dir string) error {
	c, err := contentcode.App(dir)
	if err != nil || c == nil {
		return err
	}
	if _, err := contentcode.Generate(c); err != nil {
		return fmt.Errorf("compiling the code blocks of the content: %w", err)
	}
	return nil
}
