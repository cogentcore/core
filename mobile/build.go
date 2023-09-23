// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run gendex.go -o dex.go

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"slices"

	"goki.dev/goki/config"
	"goki.dev/goki/mobile/sdkpath"
	"goki.dev/grease"
	"golang.org/x/tools/go/packages"
)

var tmpdir string

// Build compiles and encodes the app named by the import path.

// The named package must define a main function.

// The -target flag takes either android (the default), or one or more
// comma-delimited Apple platforms (TODO: apple platforms list).

// For -target android, if an AndroidManifest.xml is defined in the
// package directory, it is added to the APK output. Otherwise, a default
// manifest is generated. By default, this builds a fat APK for all supported
// instruction sets (arm, 386, amd64, arm64). A subset of instruction sets can
// be selected by specifying target type with the architecture name. E.g.
// -target=android/arm,android/386.

// For Apple -target platforms, gomobile must be run on an OS X machine with
// Xcode installed.

// By default, -target ios will generate an XCFramework for both ios
// and iossimulator. Multiple Apple targets can be specified, creating a "fat"
// XCFramework with each slice. To generate a fat XCFramework that supports
// iOS, macOS, and macCatalyst for all supportec architectures (amd64 and arm64),
// specify -target ios,macos,maccatalyst. A subset of instruction sets can be
// selectged by specifying the platform with an architecture name. E.g.
// -target=ios/arm64,maccatalyst/arm64.

// If the package directory contains an assets subdirectory, its contents
// are copied into the output.
func Build(c *config.Config) error {
	_, err := BuildImpl(c)
	return err
}

// BuildImpl builds a package for mobiles based on the given config info.
// BuildImpl returns a built package information and an error if exists.
func BuildImpl(c *config.Config) (*packages.Package, error) {
	cleanup, err := BuildEnvInit(c)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Special case to add iossimulator if we don't already have it and we have ios
	hasIOSSimulator := slices.ContainsFunc(c.Build.Target,
		func(p config.Platform) bool { return p.OS == "iossimulator" })
	hasIOS := slices.ContainsFunc(c.Build.Target,
		func(p config.Platform) bool { return p.OS == "ios" })
	if !hasIOSSimulator && hasIOS {
		c.Build.Target = append(c.Build.Target, config.Platform{OS: "iossimulator", Arch: "arm64"}) // TODO: set arch better here
	}

	// TODO(ydnar): this should work, unless build tags affect loading a single package.
	// Should we try to import packages with different build tags per platform?
	pkgs, err := packages.Load(PackagesConfig(c, &c.Build.Target[0]), c.Build.Package)
	if err != nil {
		return nil, err
	}

	// len(pkgs) can be more than 1 e.g., when the specified path includes `...`.
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("expected 1 package but got %d", len(pkgs))
	}

	pkg := pkgs[0]

	if pkg.Name != "main" && c.Build.Output != "" {
		return nil, fmt.Errorf("cannot set -o when building non-main package")
	}

	var nmpkgs map[string]bool
	switch {
	case IsAndroidPlatform(c.Build.Target[0].OS):
		if pkg.Name != "main" {
			for _, t := range c.Build.Target {
				if err := GoBuild(c, pkg.PkgPath, AndroidEnv[t.Arch]); err != nil {
					return nil, err
				}
			}
			return pkg, nil
		}
		nmpkgs, err = GoAndroidBuild(c, pkg, c.Build.Target)
		if err != nil {
			return nil, err
		}
	case IsApplePlatform(c.Build.Target[0].OS):
		if !XCodeAvailable() {
			return nil, fmt.Errorf("-target=%s requires XCode", c.Build.Target)
		}
		if pkg.Name != "main" {
			for _, t := range c.Build.Target {
				// Catalyst support requires iOS 13+
				v, _ := strconv.ParseFloat(c.Build.IOSVersion, 64)
				if t.OS == "maccatalyst" && v < 13.0 {
					return nil, errors.New("catalyst requires -iosversion=13 or higher")
				}
				if err := GoBuild(c, pkg.PkgPath, AppleEnv[t.String()]); err != nil {
					return nil, err
				}
			}
			return pkg, nil
		}
		if c.Build.BundleID == "" {
			return nil, fmt.Errorf("-target=ios requires -bundleid set")
		}
		nmpkgs, err = GoAppleBuild(c, pkg, c.Build.BundleID, c.Build.Target)
		if err != nil {
			return nil, err
		}
	}

	// This is not correctly detecting use of mobile/app.
	// Furthermore, even if it did, people should be able to use the gomobile build tool
	// to build things for mobile without using the actual mobile/app package.
	// Therefore, it has been commented out, at least temporarily.
	// TODO: decide to what to do here in the long-term.
	_ = nmpkgs
	// if !nmpkgs["goki.dev/mobile/app"] {
	// 	return nil, fmt.Errorf(`%s does not import "goki.dev/mobile/app"`, pkg.PkgPath)
	// }

	return pkg, nil
}

var NmRE = regexp.MustCompile(`[0-9a-f]{8} t _?(?:.*/vendor/)?(golang.org/x.*/[^.]*)`)

func ExtractPkgs(c *config.Config, nm string, path string) (map[string]bool, error) {
	if c.Build.PrintOnly {
		return map[string]bool{"goki.dev/mobile/app": true}, nil // TODO: fix import paths
	}
	r, w := io.Pipe()
	cmd := exec.Command(nm, path)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr

	nmpkgs := make(map[string]bool)
	errc := make(chan error, 1)
	go func() {
		s := bufio.NewScanner(r)
		for s.Scan() {
			if res := NmRE.FindStringSubmatch(s.Text()); res != nil {
				nmpkgs[res[1]] = true
			}
		}
		errc <- s.Err()
	}()

	err := cmd.Run()
	w.Close()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %v", nm, path, err)
	}
	if err := <-errc; err != nil {
		return nil, fmt.Errorf("%s %s: %v", nm, path, err)
	}
	return nmpkgs, nil
}

var Xout io.Writer = os.Stderr

func PrintCmd(format string, args ...any) {
	cmd := fmt.Sprintf(format+"\n", args...)
	if tmpdir != "" {
		cmd = strings.Replace(cmd, tmpdir, "$WORK", -1)
	}
	if androidHome, err := sdkpath.AndroidHome(); err == nil {
		cmd = strings.Replace(cmd, androidHome, "$ANDROID_HOME", -1)
	}
	if GoMobilePath != "" {
		cmd = strings.Replace(cmd, GoMobilePath, "$GOMOBILE", -1)
	}
	if gopath := GoEnv("GOPATH"); gopath != "" {
		cmd = strings.Replace(cmd, gopath, "$GOPATH", -1)
	}
	if env := os.Getenv("HOMEPATH"); env != "" {
		cmd = strings.Replace(cmd, env, "$HOMEPATH", -1)
	}
	fmt.Fprint(Xout, cmd)
}

func GoBuild(c *config.Config, src string, env []string, args ...string) error {
	return GoCmd(c, "build", []string{src}, env, args...)
}

func GoBuildAt(c *config.Config, at string, src string, env []string, args ...string) error {
	return GoCmdAt(c, at, "build", []string{src}, env, args...)
}

func GoInstall(c *config.Config, srcs []string, env []string, args ...string) error {
	return GoCmd(c, "install", srcs, env, args...)
}

func GoCmd(c *config.Config, subcmd string, srcs []string, env []string, args ...string) error {
	return GoCmdAt(c, "", subcmd, srcs, env, args...)
}

func GoCmdAt(c *config.Config, at string, subcmd string, srcs []string, env []string, args ...string) error {
	cmd := exec.Command("go", subcmd)
	tags := c.Build.Tags
	if len(tags) > 0 {
		cmd.Args = append(cmd.Args, "-tags", strings.Join(tags, ","))
	}
	if grease.Verbose {
		cmd.Args = append(cmd.Args, "-v")
	}
	if subcmd != "install" && c.Build.Install {
		cmd.Args = append(cmd.Args, "-i")
	}
	if c.Build.Print {
		cmd.Args = append(cmd.Args, "-x")
	}
	if len(c.Build.GCFlags) != 0 {
		cmd.Args = append(cmd.Args, "-gcflags", strings.Join(c.Build.GCFlags, ","))
	}
	if len(c.Build.LDFlags) != 0 {
		cmd.Args = append(cmd.Args, "-ldflags", strings.Join(c.Build.LDFlags, ","))
	}
	if c.Build.Trimpath {
		cmd.Args = append(cmd.Args, "-trimpath")
	}
	if c.Build.Work {
		cmd.Args = append(cmd.Args, "-work")
	}
	cmd.Args = append(cmd.Args, args...)
	cmd.Args = append(cmd.Args, srcs...)

	// Specify GOMODCACHE explicitly. The default cache path is GOPATH[0]/pkg/mod,
	// but the path varies when GOPATH is specified at env, which results in cold cache.
	if gmc, err := GoModCachePath(); err == nil {
		env = append([]string{"GOMODCACHE=" + gmc}, env...)
	} else {
		env = append([]string{}, env...)
	}
	cmd.Env = env
	cmd.Dir = at
	return RunCmd(c, cmd)
}

func GoModTidyAt(c *config.Config, at string, env []string) error {
	cmd := exec.Command("go", "mod", "tidy")
	if grease.Verbose {
		cmd.Args = append(cmd.Args, "-v")
	}

	// Specify GOMODCACHE explicitly. The default cache path is GOPATH[0]/pkg/mod,
	// but the path varies when GOPATH is specified at env, which results in cold cache.
	if gmc, err := GoModCachePath(); err == nil {
		env = append([]string{"GOMODCACHE=" + gmc}, env...)
	} else {
		env = append([]string{}, env...)
	}
	cmd.Env = env
	cmd.Dir = at
	return RunCmd(c, cmd)
}

func GoModCachePath() (string, error) {
	out, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
