// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"strings"
	"testing"
)

// TestLinkerFlagsID tests that the linker flags set the ID of the app,
// as for its data directory on Android before main.
func TestLinkerFlagsID(t *testing.T) {
	c := &Config{ID: "com.cogent.lists"}
	if f := LinkerFlags(c); !strings.Contains(f, "-X 'cogentcore.org/core/system.AppID=com.cogent.lists'") {
		t.Errorf("no AppID in %q", f)
	}
}
