// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run gendex.go -o dex.go

package mobile

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

	"log/slog"

	"maps"

	"cogentcore.org/core/core/config"
	"cogentcore.org/core/grog"
	"cogentcore.org/core/xe"
	"golang.org/x/tools/go/packages"
)

var TmpDir string

// Build compiles and encodes the app named by the import path.
//
// The named package must define a main function.
//
// The -target flag takes either android (the default), or one or more
// comma-delimited Apple platforms (TODO: apple platforms list).
//
// For -target android, if an AndroidManifest.xml is defined in the
// package directory, it is added to the APK output. Otherwise, a default
// manifest is generated. By default, this builds a fat APK for all supported
// instruction sets (arm, 386, amd64, arm64). A subset of instruction sets can
// be selected by specifying target type with the architecture name. E.g.
// -target=android/arm,android/386.
//
// For Apple -target platforms, gomobile must be run on an OS X machine with
// Xcode installed.
//
// By default, -target ios will generate an XCFramework for both ios
// and iossimulator. Multiple Apple targets can be specified, creating a "fat"
// XCFramework with each slice. To generate a fat XCFramework that supports
// iOS, macOS, and macCatalyst for all supportec architectures (amd64 and arm64),
// specify -target ios,macos,maccatalyst. A subset of instruction sets can be
// selectged by specifying the platform with an architecture name. E.g.
// -target=ios/arm64,maccatalyst/arm64.
//
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

	for _, platform := range c.Build.Target {
		if platform.Arch == "*" {
			archs := config.ArchsForOS[platform.OS]
			c.Build.Target = make([]config.Platform, len(archs))
			for i, arch := range archs {
				c.Build.Target[i] = config.Platform{OS: platform.OS, Arch: arch}
			}
		}
	}

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
	pkgs, err := packages.Load(PackagesConfig(c, &c.Build.Target[0]), ".")
	if err != nil {
		return nil, err
	}

	// len(pkgs) can be more than 1 e.g., when the specified path includes `...`.
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("expected 1 package but got %d", len(pkgs))
	}

	pkg := pkgs[0]

	if pkg.Name != "main" {
		return nil, fmt.Errorf("cannot build non-main package")
	}

	if c.ID == "" {
		return nil, fmt.Errorf("id must be set when building for mobile")
	}

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
		_, err = GoAndroidBuild(c, pkg, c.Build.Target)
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
		_, err = GoAppleBuild(c, pkg, c.Build.Target)
		if err != nil {
			return nil, err
		}
	}

	return pkg, nil
}

var NmRE = regexp.MustCompile(`[0-9a-f]{8} t _?(?:.*/vendor/)?(golang.org/x.*/[^.]*)`)

func ExtractPkgs(c *config.Config, nm string, path string) (map[string]bool, error) {
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

func GoBuild(c *config.Config, src string, env map[string]string, args ...string) error {
	return GoCmd(c, "build", []string{src}, env, args...)
}

func GoBuildAt(c *config.Config, at string, src string, env map[string]string, args ...string) error {
	return GoCmdAt(c, at, "build", []string{src}, env, args...)
}

func GoInstall(c *config.Config, srcs []string, env map[string]string, args ...string) error {
	return GoCmd(c, "install", srcs, env, args...)
}

func GoCmd(c *config.Config, subcmd string, srcs []string, env map[string]string, args ...string) error {
	return GoCmdAt(c, "", subcmd, srcs, env, args...)
}

func GoCmdAt(c *config.Config, at string, subcmd string, srcs []string, env map[string]string, args ...string) error {
	cargs := []string{subcmd}
	// cmd := exec.Command("go", subcmd)
	var tags []string
	if c.Build.Debug {
		tags = append(tags, "debug")
	}
	if len(tags) > 0 {
		cargs = append(cargs, "-tags", strings.Join(tags, ","))
	}
	if grog.UserLevel <= slog.LevelInfo {
		cargs = append(cargs, "-v")
	}
	cargs = append(cargs, args...)
	cargs = append(cargs, srcs...)

	xc := xe.Major().SetDir(at)
	maps.Copy(xc.Env, env)

	// Specify GOMODCACHE explicitly. The default cache path is GOPATH[0]/pkg/mod,
	// but the path varies when GOPATH is specified at env, which results in cold cache.
	if gmc, err := GoModCachePath(); err == nil {
		xc.SetEnv("GOMODCACHE", gmc)
	}
	return xc.Run("go", cargs...)
}

func GoModTidyAt(c *config.Config, at string, env map[string]string) error {
	args := []string{"mod", "tidy"}
	if grog.UserLevel <= slog.LevelInfo {
		args = append(args, "-v")
	}
	xc := xe.Major().SetDir(at)
	maps.Copy(xc.Env, env)

	// Specify GOMODCACHE explicitly. The default cache path is GOPATH[0]/pkg/mod,
	// but the path varies when GOPATH is specified at env, which results in cold cache.
	if gmc, err := GoModCachePath(); err == nil {
		xc.SetEnv("GOMODCACHE", gmc)
	}
	return xe.Run("go", args...)
}

func GoModCachePath() (string, error) {
	out, err := xe.Output("go", "env", "GOMODCACHE")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
