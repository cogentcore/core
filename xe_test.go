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
	m := Main()
	err := RunSh(m, "go version")
	if err != nil {
		t.Error(err)
	}
	err = RunSh(m, "git version")
	if err != nil {
		t.Error(err)
	}
	err = RunSh(m, "echo hello")
	if err != nil {
		t.Error(err)
	}
	err = RunSh(m, "go bild")
	if err == nil { // we want it to fail
		t.Error("expected error but got none")
	}
}
