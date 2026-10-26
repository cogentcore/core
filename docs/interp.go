// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !nointerp

package main

import (
	"cogentcore.org/core/content"
	"cogentcore.org/core/mvmcore"
)

// The code blocks of the docs are compiled into the app (contentcode_gen.go).
// The mvm interpreter runs code that users edit, except with the nointerp
// tag. In js builds, mvmcore.BindTextEditor is a deferred function (see
// core.toml): mvm is loaded when code is first edited.

func init() {
	content.BindTextEditor = mvmcore.BindTextEditor
}
