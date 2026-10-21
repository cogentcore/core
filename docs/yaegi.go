// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !noyaegi

package main

import (
	"cogentcore.org/core/content"
	"cogentcore.org/core/yaegicore"
)

// The code blocks of the docs are compiled into the app (contentcode_gen.go).
// The yaegi interpreter runs code that users edit, except with the noyaegi
// tag. In js builds, yaegicore.BindTextEditor is a deferred function (see
// core.toml): yaegi is loaded when code is first edited.

func init() {
	content.BindTextEditor = yaegicore.BindTextEditor
}
