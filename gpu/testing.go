// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"os"
	"sync"
)

// Skipper is the subset of [testing.TB] that [SkipIfNoGPU] uses.
type Skipper interface {
	Helper()
	Skip(args ...any)
	Fatal(args ...any)
}

var (
	availableOnce sync.Once
	available     bool
)

// Available returns whether WebGPU has at least one adapter on this
// machine, which a CI runner without a GPU (hardware or software)
// does not have. It is computed once.
func Available() bool {
	availableOnce.Do(func() {
		inst := Instance()
		available = inst != nil && len(inst.EnumerateAdapters(nil)) > 0
	})
	return available
}

// SkipIfNoGPU skips the test if there is no GPU adapter (see [Available]).
// If the CORE_REQUIRE_GPU environment variable is set, as on a CI runner
// that has a software GPU, it fails the test instead, so that a missing
// GPU is not silently skipped.
func SkipIfNoGPU(t Skipper) {
	t.Helper()
	if Available() {
		return
	}
	if os.Getenv("CORE_REQUIRE_GPU") != "" {
		t.Fatal("no GPU adapter is available, and CORE_REQUIRE_GPU is set")
	}
	t.Skip("no GPU adapter is available")
}
