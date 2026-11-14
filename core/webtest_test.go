// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"runtime"
	"testing"
)

// skipOnWeb skips the test on the web (see goscript_test.go), as one that
// does not apply there, for the given reason, such as that it uses the
// files of the operating system, or more than one window.
func skipOnWeb(t *testing.T, reason string) {
	t.Helper()
	if runtime.GOOS == "js" {
		t.Skip("not on the web: " + reason)
	}
}
