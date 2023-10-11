// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grog

import "log/slog"

// UserLevel is the verbosity [slog.Level] that the user has selected for
// what logging and printing messages should be shown. Messages at
// levels at or above this level will be shown. It should typically
// be set through xe to the end user's preference. The default user
// verbosity level is [slog.LevelInfo]. If the build tag "debug" is
// specified, it is [slog.LevelDebug]. If the build tag "release" is
// specified, it is [slog.levelWarn]. Any updates to this value will
// be automatically reflected in the behavior of the grog default logger.
var UserLevel = defaultUserLevel

// LevelFromFlags returns the [slog.Level] object corresponding to the given
// user flag options. The flags correspond to the following values:
//   - vv: [slog.LevelDebug]
//   - v: [slog.LevelInfo]
//   - q: [slog.LevelError]
//   - (default: same as the default for [UserLevel])
//
// The flags are evaluated in that order, so, for example, if both
// vv and q are specified, it will still return [Debug].
func LevelFromFlags(vv, v, q bool) slog.Level {
	switch {
	case vv:
		return slog.LevelDebug
	case v:
		return slog.LevelInfo
	case q:
		return slog.LevelError
	default:
		return defaultUserLevel
	}
}
