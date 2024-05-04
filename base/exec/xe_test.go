// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"log/slog"
	"testing"

	"cogentcore.org/core/base/logx"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	logx.UserLevel = slog.LevelInfo
	xc := Major().SetFatal(true)
	xc.Run("go", "version")
	xc.Run("git", "version")
	xc.Run("echo", " hello")
}

func TestError(t *testing.T) {
	assert.Error(t, Run("go", "bild"))
}
