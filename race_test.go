// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// racePackages are the packages whose tests [TestRace] runs with the race
// detector: the main GUI packages and the packages they use for their state,
// which run code in the event loop, render and background goroutines.
// It does not include packages that need a GPU or a display, or text/tex,
// whose tests are too slow with the race detector.
var racePackages = []string{
	"./core",
	"./content",
	"./htmlcore",
	"./filetree",
	"./tree",
	"./events/...",
	"./styles/...",
	"./paint",
	"./text/lines",
	"./text/textcore",
	"./text/highlighting",
	"./text/rich",
	"./text/shaped/...",
	"./enums/...",
	"./system/driver/offscreen",
}

// raceRegexp matches the start of a data race report of the race detector.
var raceRegexp = regexp.MustCompile(`(?m)^WARNING: DATA RACE$`)

// raceFailRegexp matches the names of failing tests in the output of go test.
var raceFailRegexp = regexp.MustCompile(`(?m)^\s*--- FAIL: (\S+)`)

// TestRace runs the tests of [racePackages] with the race detector, and
// fails if it reports any data races. It takes about a minute, so it is
// skipped in short mode (as in CI). Other test failures are logged but do
// not fail it, as some tests that compare rendered images are sensitive to
// the slower timing with the race detector.
func TestRace(t *testing.T) {
	if testing.Short() {
		t.Skip("the race detector tests are skipped in short mode")
	}
	args := append([]string{"test", "-race", "-count=1", "-timeout=30m"}, racePackages...)
	cmd := exec.Command("go", args...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	sout := string(out)
	for _, m := range raceFailRegexp.FindAllStringSubmatch(sout, -1) {
		t.Logf("failing test with the race detector: %s", m[1])
	}
	races := raceRegexp.FindAllStringIndex(sout, -1)
	if len(races) == 0 {
		if err != nil && !strings.Contains(sout, "--- FAIL") {
			t.Fatalf("go test -race failed: %v\n%s", err, sout)
		}
		return
	}
	// Each report ends with a line of = characters.
	for _, r := range races {
		report := sout[r[0]:]
		if end := strings.Index(report, "\n==================\n"); end >= 0 {
			report = report[:end]
		}
		t.Errorf("data race:\n%s", report)
	}
	t.Errorf("the race detector reported %d data races", len(races))
}
