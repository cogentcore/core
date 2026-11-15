// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"cmp"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/goscript"
	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/cmd/config"
	"cogentcore.org/core/cmd/web"
)

// Test runs the tests of the given packages for the target platform, as
// with go test. For the js platform (core test js), the tests are compiled
// to JavaScript by GoScript, with the build tags and overrides of core
// build js, and run in bun, or in Chrome for the packages that say so
// with a //core:js-test line in a test file (see the goscript package in
// base), which also has other options. A package pattern with ... only
// matches the packages that have such a line. The tools are installed
// first if needed (as by core setup -js).
func Test(c *config.Config) error { //types:add
	if len(c.Build.Target) != 1 {
		return fmt.Errorf("expected 1 target platform, but got %d (%v)", len(c.Build.Target), c.Build.Target)
	}
	pkgs := c.Test.Packages
	if len(pkgs) == 0 {
		pkgs = []string{"."}
	}
	verbose := logx.UserLevel <= slog.LevelInfo
	if t := c.Build.Target[0]; t.OS != "js" {
		// the other platforms are those of go test
		args := []string{"test"}
		if c.Test.Run != "" {
			args = append(args, "-run", c.Test.Run)
		}
		if c.Test.Bench != "" {
			args = append(args, "-bench", c.Test.Bench)
		}
		if c.Test.BenchTime != "" {
			args = append(args, "-benchtime", c.Test.BenchTime)
		}
		if verbose {
			args = append(args, "-v")
		}
		args = append(args, pkgs...)
		return exec.Verbose().SetBuffer(false).Run("go", args...)
	}
	if err := web.SetupJS(); err != nil {
		return err
	}
	dirs, err := testPackageDirs(pkgs)
	if err != nil {
		return err
	}

	// each package is tested in turn, in each run that it has
	var failed []string
	for _, d := range dirs {
		runs, err := goscript.PackageOptions(d.dir)
		if err != nil {
			return err
		}
		if len(runs) == 0 {
			if d.pattern {
				continue // not opted in
			}
			runs = []goscript.Options{{}}
		}
		for _, o := range runs {
			o.Tags = append(strings.Split(web.JSTags, ","), o.Tags...)
			o.Run = cmp.Or(c.Test.Run, o.Run)
			o.Bench, o.BenchTime = c.Test.Bench, c.Test.BenchTime
			o.Browser = o.Browser || c.Test.Browser
			if o.Browser {
				o.BrowserChannel = "chrome"
			}
			o.Verbose = verbose
			var out io.Writer = io.Discard
			if verbose || o.Bench != "" {
				out = os.Stdout
			}
			var buf strings.Builder
			if out == io.Discard {
				out = &buf // shown if it fails
			}
			st := time.Now()
			if err := goscript.RunTests(d.dir, o, out); err != nil {
				fmt.Print(buf.String())
				fmt.Printf("FAIL\t%s\t%v: %v\n", d.importPath, time.Since(st).Round(time.Millisecond), err)
				failed = append(failed, d.importPath)
				continue
			}
			fmt.Printf("ok  \t%s\t%v\n", d.importPath, time.Since(st).Round(time.Millisecond))
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("the tests of %d package(s) failed: %s", len(failed), strings.Join(failed, ", "))
	}
	return nil
}

// testPackage is a package to test, for [Test].
type testPackage struct {
	importPath, dir string

	// pattern is whether the package was matched by a pattern with ...
	// in it, and not given itself.
	pattern bool
}

// testPackageDirs returns the packages with tests of the given package
// patterns.
func testPackageDirs(patterns []string) ([]testPackage, error) {
	var pkgs []testPackage
	seen := map[string]bool{}
	for _, pat := range patterns {
		out, err := exec.Minor().Output("go", "list", "-e", "-f", "{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}} {{.Dir}}{{end}}", pat)
		if err != nil {
			return nil, err
		}
		for line := range strings.SplitSeq(out, "\n") {
			ip, dir, ok := strings.Cut(strings.TrimSpace(line), " ")
			if !ok || seen[dir] {
				continue
			}
			seen[dir] = true
			pkgs = append(pkgs, testPackage{importPath: ip, dir: dir, pattern: strings.Contains(pat, "...")})
		}
	}
	return pkgs, nil
}
