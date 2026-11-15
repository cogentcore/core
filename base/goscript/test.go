// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !goscript

package goscript

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// EnvVar is the environment variable that must be set to a true value
// (such as 1) for [Test] to run. Otherwise, [Test] is skipped, because
// compiling and running the tests in JavaScript takes significant time,
// and requires goscript and bun (https://bun.sh) to be installed.
const EnvVar = "CORE_GOSCRIPT_TEST"

// BenchEnvVar is the environment variable that, when set to a regular
// expression, makes [Test] also run the benchmarks matching it in JavaScript,
// as with go test -bench, logging their results (use go test -v to see them).
// The tests are not run in this case, unless [Options.Run] is set.
const BenchEnvVar = "CORE_GOSCRIPT_BENCH"

// BenchTimeEnvVar is the environment variable that sets the minimum time to
// run each benchmark when [BenchEnvVar] is set, as a duration such as 2s.
const BenchTimeEnvVar = "CORE_GOSCRIPT_BENCHTIME"

// benchRegexp matches benchmark result lines in the output of goscript test.
var benchRegexp = regexp.MustCompile(`(?m)^Benchmark\S*\t.*$`)

// Options are the options for [Test].
type Options struct {

	// Run, if non-empty, only runs the tests matching this regular expression.
	Run string

	// Timeout is the maximum time for compiling and running the tests.
	// It defaults to 5 minutes.
	Timeout time.Duration

	// Browser runs the tests in a Chromium browser instead of bun.
	Browser bool

	// BrowserChannel, if non-empty, runs the [Options.Browser] tests in
	// this installed browser, such as "chrome", instead of the Chromium
	// that Playwright downloads.
	BrowserChannel string

	// BrowserArgs are additional command-line arguments for the browser,
	// such as "--enable-unsafe-webgpu" to enable WebGPU in headless Chrome.
	BrowserArgs []string

	// Tags are additional build tags (the goscript tag is always added).
	Tags []string

	// SkipTypecheck skips the TypeScript type checking of the compiled
	// code, which goscript test otherwise does before running it, as core
	// build js does (bun only strips the types). It is for packages whose
	// compiled code has type errors that do not affect what they do, so
	// that their tests still run (as mvmcore, whose mvm interpreter the
	// docs use on the web).
	SkipTypecheck bool
}

// failRegexp matches the names of failing tests in the output of goscript test.
var failRegexp = regexp.MustCompile(`(?m)^\s*--- FAIL: (\S+)`)

// Test runs the Go tests of the package in the current directory compiled to
// JavaScript by goscript, and reports any failures as failures of t. It is
// skipped unless [EnvVar] is set. It must be called from a test in a file with
// a !goscript build constraint, so that it does not run itself recursively
// (see the package documentation for an example).
func Test(t *testing.T, opts ...Options) {
	t.Helper()
	if enabled, _ := strconv.ParseBool(os.Getenv(EnvVar)); !enabled {
		t.Skipf("set %s=1 to run the tests compiled to JavaScript by goscript", EnvVar)
	}
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Timeout == 0 {
		o.Timeout = 5 * time.Minute
	}
	gs, err := exec.LookPath("goscript")
	if err != nil {
		t.Fatal("goscript must be installed: go install github.com/s4wave/goscript/cmd/goscript@latest")
	}
	if _, err := exec.LookPath("bun"); err != nil {
		t.Fatal("bun must be installed: see https://bun.sh")
	}

	bench := os.Getenv(BenchEnvVar)
	if bench != "" && o.Timeout < 30*time.Minute {
		o.Timeout = 30 * time.Minute
	}
	args := []string{"test", "--workdir", t.TempDir(), "--timeout", o.Timeout.String()}
	if o.Run != "" {
		args = append(args, "--run", o.Run)
	} else if bench != "" {
		args = append(args, "--run", "^$")
	}
	if bench != "" {
		args = append(args, "--bench", bench)
		if bt := os.Getenv(BenchTimeEnvVar); bt != "" {
			args = append(args, "--benchtime", bt)
		}
	}
	if o.Browser {
		args = append(args, "--browser")
		if o.BrowserChannel != "" {
			args = append(args, "--browser-channel", o.BrowserChannel)
		}
		for _, a := range o.BrowserArgs {
			args = append(args, "--browser-arg="+a)
		}
	}
	if len(o.Tags) > 0 {
		args = append(args, "--tags", strings.Join(o.Tags, ","))
	}
	// the goscript overrides of the dependencies, as core build js uses
	// (such as those of the mvm interpreter)
	overrides, err := overrideDirs(append([]string{"goscript"}, o.Tags...))
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range overrides {
		args = append(args, "--gs-path", dir)
	}
	if testing.Verbose() {
		args = append(args, "-v")
	}
	args = append(args, ".")

	cmd := exec.Command(gs, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "FORCE_COLOR=0")
	if o.SkipTypecheck {
		// goscript type checks with the tsrs it finds first on the PATH,
		// so one that checks nothing skips it
		bin := t.TempDir()
		noop := filepath.Join(bin, "tsrs")
		script := "#!/bin/sh\nexit 0\n"
		if runtime.GOOS == "windows" {
			noop += ".cmd"
			script = "@exit /b 0\r\n"
		}
		if err := os.WriteFile(noop, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		cmd.Env = append(cmd.Env, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	// go test does not pass GOROOT to tests as of Go 1.27, but the tests
	// compiled to JavaScript get runtime.GOROOT from the environment.
	if os.Getenv("GOROOT") == "" {
		if goroot, err := exec.Command("go", "env", "GOROOT").Output(); err == nil {
			cmd.Env = append(cmd.Env, "GOROOT="+strings.TrimSpace(string(goroot)))
		}
	}
	out, err := cmd.CombinedOutput()
	if err == nil {
		if testing.Verbose() {
			t.Log(string(out))
		} else if bench != "" {
			t.Log("\n" + strings.Join(benchRegexp.FindAllString(string(out), -1), "\n"))
		}
		return
	}
	failed := []string{}
	for _, m := range failRegexp.FindAllStringSubmatch(string(out), -1) {
		failed = append(failed, m[1])
	}
	msg := "goscript test failed"
	if len(failed) > 0 {
		msg = fmt.Sprintf("goscript tests failed: %s", strings.Join(failed, ", "))
	}
	t.Errorf("%s (%v):\n%s", msg, err, out)
}

// overrideDirs returns the gsoverride directories of the modules of the
// dependencies of the package in the current directory, built with the
// given tags for js, as core build js does (see cmd/web).
func overrideDirs(tags []string) ([]string, error) {
	cmd := exec.Command("go", "list", "-deps", "-tags", strings.Join(tags, ","), "-f", "{{with .Module}}{{.Dir}}{{end}}", ".")
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("goscript: listing the modules of the dependencies: %w", err)
	}
	var dirs []string
	seen := map[string]bool{}
	for dir := range strings.SplitSeq(string(out), "\n") {
		dir = strings.TrimSpace(dir)
		if dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		gs := filepath.Join(dir, "gsoverride")
		if info, err := os.Stat(gs); err == nil && info.IsDir() {
			dirs = append(dirs, gs)
		}
	}
	return dirs, nil
}
