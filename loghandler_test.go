// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grog

import (
	"log"
	"log/slog"
	"testing"
)

func TestDefaultLogger(t *testing.T) {
	UserLevel = slog.LevelDebug

	slog.Debug("this is debug")
	slog.Info("this is info")
	slog.Warn("this is warn")
	slog.Error("this is error\n")

	log.Println("this is standard log")

	PrintDebug("\nthis is PrintDebug\n")
	PrintlnInfo("this is PrintlnInfo")
	PrintlnWarn("this is PrintlnWarn")
	PrintfError("this is %q", "PrintfError")
}
