// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !debug && !release

package logx

import "log/slog"

var defaultUserLevel = slog.LevelInfo
