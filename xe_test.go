// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xe

import (
	"log/slog"
	"testing"

	"goki.dev/grog"
)

func TestRun(t *testing.T) {
	grog.UserLevel = slog.LevelWarn
	xc := Major().SetFatal(true)
	xc.RunSh("go version")
	xc.RunSh("git version")
	xc.RunSh("echo hello")

	err := Major().RunSh("go bild")
	if err == nil { // we want it to fail
		t.Error("expected error but got none")
	}
}
