// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js || ios

package filetree

import "context"

// watchDirs does nothing on platforms without file system watching.
func watchDirs(_ context.Context, _ []string, _ chan struct{}) (*watchBackend, error) {
	return nil, nil
}
