// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"

	"cogentcore.org/core/content"
)

func TestCodeKey(t *testing.T) {
	for _, src := range []string{"", "core.NewButton(b)\n", "\tx := 1\n  y := \"é ü\"\n", "func main() {\n}\n"} {
		if got, want := codeKey(src), content.CodeKey(src); got != want {
			t.Errorf("codeKey(%q) = %q, content.CodeKey = %q", src, got, want)
		}
	}
	if codeKey("a b\n\tc") != codeKey("abc") {
		t.Error("whitespace changes the key")
	}
}
