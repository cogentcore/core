// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xe

import "testing"

func TestRun(t *testing.T) {
	cfg := DefaultConfig()
	RunSh(cfg, "go version")
	RunSh(cfg, "git version")
	RunSh(cfg, "echo hello")
}
