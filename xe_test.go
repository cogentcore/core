// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xe

import "testing"

func TestRun(t *testing.T) {
	cfg := DefaultConfig()
	Run(cfg, "echo", "hello")
}
