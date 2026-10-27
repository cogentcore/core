// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js || ios

package core

import "context"

// watchFilePickerDir does nothing on platforms without file system watching.
func watchFilePickerDir(ctx context.Context, dir string, changed func()) error {
	return nil
}
