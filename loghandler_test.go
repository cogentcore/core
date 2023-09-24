// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grog

import (
	"log/slog"
	"testing"
)

func TestDefaultLogger(t *testing.T) {
	UserLevel = Debug
	SetDefaultLogger()

	slog.Debug("this is debug")
	slog.Info("this is info")
	slog.Warn("this is warn")
}
