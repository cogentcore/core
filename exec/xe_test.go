// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"log/slog"
	"testing"

	"cogentcore.org/core/logx"
)

func TestRun(t *testing.T) {
	logx.UserLevel = slog.LevelInfo
	xc := Major().SetFatal(true)
	xc.RunSh("go version")
	xc.RunSh("git version")
	xc.RunSh("echo hello")
}

func TestError(t *testing.T) {
	err := Major().RunSh("go bild")
	if err == nil { // we want it to fail
		t.Error("expected error but got none")
	}
}
