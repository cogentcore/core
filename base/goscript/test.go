// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !goscript

package goscript

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Directive is the comment directive with which a package opts in to
// having its tests run compiled to JavaScript by core test js for a
// package pattern with ... in it (see [PackageOptions]). It is a line
// comment in any test file of the package, optionally followed by options:
//
//	//core:js-test browser timeout=20m
//
// Each such line is one run of the tests, with these options:
//   - browser: run the tests in Chrome instead of bun ([Options.Browser]).
//   - browser-arg=ARG: an argument for the browser ([Options.BrowserArgs]).
//   - run=REGEXP: only run the matching tests ([Options.Run]).
//   - timeout=DURATION: the timeout of the run ([Options.Timeout]).
//   - skip-typecheck: skip the type checking ([Options.SkipTypecheck]).
//   - tags=TAG,...: additional build tags ([Options.Tags]).
const Directive = "//core:js-test"

// Options are the options for [RunTests].
type Options struct {

	// Run, if non-empty, only runs the tests matching this regular expression.
	Run string

	// Bench, if non-empty, also runs the benchmarks matching this regular
	// expression, as with go test -bench. The tests are not run in this
	// case, unless [Options.Run] is set.
	Bench string

	// BenchTime is the minimum time to run each benchmark, as a duration
	// such as 2s, if [Options.Bench] is set.
	BenchTime string

	// Timeout is the maximum time for compiling and running the tests.
	// It defaults to 5 minutes (30 minutes with [Options.Bench]).
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

	// Verbose makes goscript test print the output of all of the tests,
	// as with go test -v.
	Verbose bool
}

// RunTests runs the Go tests of the package in the given directory compiled
// to JavaScript by goscript test, writing its output to the given writer.
// It returns an error if the tests could not be run or did not pass, which
// names the failing tests. goscript and bun (https://bun.sh) must be
// installed (as by core setup -js), and tsrs, unless [Options.SkipTypecheck].
func RunTests(dir string, o Options, out io.Writer) error {
	if o.Timeout == 0 {
		o.Timeout = 5 * time.Minute
		if o.Bench != "" {
			o.Timeout = 30 * time.Minute
		}
	}
	gs, err := exec.LookPath("goscript")
	if err != nil {
		return fmt.Errorf("goscript must be installed (core setup -js): %w", err)
	}
	if _, err := exec.LookPath("bun"); err != nil {
		return fmt.Errorf("bun must be installed (core setup -js): %w", err)
	}
	work, err := os.MkdirTemp("", "goscript-test-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	// the options of goscript test
	args := []string{"test", "--workdir", work, "--timeout", o.Timeout.String()}
	if o.Run != "" {
		args = append(args, "--run", o.Run)
	} else if o.Bench != "" {
		args = append(args, "--run", "^$")
	}
	if o.Bench != "" {
		args = append(args, "--bench", o.Bench)
		if o.BenchTime != "" {
			args = append(args, "--benchtime", o.BenchTime)
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
	overrides, err := overrideDirs(dir, append([]string{"goscript"}, o.Tags...))
	if err != nil {
		return err
	}
	for _, od := range overrides {
		args = append(args, "--gs-path", od)
	}
	if o.Verbose {
		args = append(args, "-v")
	}
	args = append(args, ".")

	// the environment of goscript test
	cmd := exec.Command(gs, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "FORCE_COLOR=0")
	if o.SkipTypecheck {
		// goscript type checks with the tsrs it finds first on the PATH,
		// so one that checks nothing skips it
		bin := filepath.Join(work, "bin")
		noop := filepath.Join(bin, "tsrs")
		script := "#!/bin/sh\nexit 0\n"
		if runtime.GOOS == "windows" {
			noop += ".cmd"
			script = "@exit /b 0\r\n"
		}
		if err := os.MkdirAll(bin, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(noop, []byte(script), 0o755); err != nil {
			return err
		}
		cmd.Env = append(cmd.Env, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	// the tests compiled to JavaScript get runtime.GOROOT from the environment
	if os.Getenv("GOROOT") == "" {
		if goroot, err := exec.Command("go", "env", "GOROOT").Output(); err == nil {
			cmd.Env = append(cmd.Env, "GOROOT="+strings.TrimSpace(string(goroot)))
		}
	}

	// the output is also kept to find the failing tests
	var buf strings.Builder
	cmd.Stdout = io.MultiWriter(out, &buf)
	cmd.Stderr = cmd.Stdout
	if err := cmd.Run(); err != nil {
		var failed []string
		for _, m := range failRegexp.FindAllStringSubmatch(buf.String(), -1) {
			failed = append(failed, m[1])
		}
		if len(failed) > 0 {
			return fmt.Errorf("goscript tests failed: %s (%w)", strings.Join(failed, ", "), err)
		}
		return fmt.Errorf("goscript test failed: %w", err)
	}
	return nil
}

// failRegexp matches the names of failing tests in the output of goscript test.
var failRegexp = regexp.MustCompile(`(?m)^\s*--- FAIL: (\S+)`)

// PackageOptions returns the options of the runs of the tests of the
// package in the given directory that its test files give with [Directive]
// lines, in the order of the files and lines (none if it has no such line).
func PackageOptions(dir string) ([]Options, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return nil, err
	}
	var opts []Options
	for _, fn := range files {
		f, err := os.Open(fn)
		if err != nil {
			return nil, err
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			rest, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), Directive)
			if !ok || (rest != "" && rest[0] != ' ' && rest[0] != '\t') {
				continue
			}
			o, err := parseDirective(rest)
			if err != nil {
				f.Close()
				return nil, fmt.Errorf("%s: %w", fn, err)
			}
			opts = append(opts, o)
		}
		f.Close()
		if err := sc.Err(); err != nil {
			return nil, err
		}
	}
	return opts, nil
}

// parseDirective returns the options of the given fields of a [Directive].
func parseDirective(s string) (Options, error) {
	var o Options
	for _, fld := range strings.Fields(s) {
		name, value, _ := strings.Cut(fld, "=")
		switch name {
		case "browser":
			o.Browser = true
		case "browser-arg":
			o.BrowserArgs = append(o.BrowserArgs, value)
		case "run":
			o.Run = value
		case "timeout":
			d, err := time.ParseDuration(value)
			if err != nil {
				return o, fmt.Errorf("%s timeout: %w", Directive, err)
			}
			o.Timeout = d
		case "skip-typecheck":
			o.SkipTypecheck = true
		case "tags":
			o.Tags = append(o.Tags, strings.Split(value, ",")...)
		default:
			return o, fmt.Errorf("%s: unknown option %q", Directive, fld)
		}
	}
	return o, nil
}

// overrideDirs returns the gsoverride directories of the modules of the
// dependencies of the package in the given directory, built with the
// given tags for js, as core build js does (see cmd/web).
func overrideDirs(dir string, tags []string) ([]string, error) {
	cmd := exec.Command("go", "list", "-deps", "-tags", strings.Join(tags, ","), "-f", "{{with .Module}}{{.Dir}}{{end}}", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("goscript: listing the modules of the dependencies: %w", err)
	}
	var dirs []string
	seen := map[string]bool{}
	for d := range strings.SplitSeq(string(out), "\n") {
		d = strings.TrimSpace(d)
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		gs := filepath.Join(d, "gsoverride")
		if info, err := os.Stat(gs); err == nil && info.IsDir() {
			dirs = append(dirs, gs)
		}
	}
	return dirs, nil
}
