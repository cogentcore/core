// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/cmd/config"
	"cogentcore.org/core/cmd/generate"
	"cogentcore.org/core/cmd/mobile"
)

// Install installs the package on the local system.
// It uses the same config info as build.
// lsregister is the path of the macOS tool that registers apps
// with Launch Services.
const lsregister = "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

func Install(c *config.Config) error { //types:add
	for i, p := range c.Build.Target {
		err := config.OSSupported(p.OS)
		if err != nil {
			return fmt.Errorf("install: %w", err)
		}
		if p.Arch == "*" {
			if p.OS == "android" || p.OS == "ios" {
				p.Arch = "arm64"
			} else {
				p.Arch = runtime.GOARCH
			}
			c.Build.Target[i] = p
		}

		switch p.OS {
		case "android", "ios":
			err := Build(c)
			if err != nil {
				return fmt.Errorf("error building: %w", err)
			}
			// we only want this target for install
			ot := c.Build.Target
			c.Build.Target = []config.Platform{p}
			err = mobile.Install(c)
			c.Build.Target = ot
			if err != nil {
				return fmt.Errorf("install: %w", err)
			}
		case "wasm", "js":
			return fmt.Errorf("can not install on platform %s; use build or run instead", p.OS)
		case "darwin":
			c.Pack.DMG = false
			err := Pack(c)
			if err != nil {
				return err
			}
			app := filepath.Join("/Applications", c.Name+".app")
			if err := removeExecutable(appExecutable(c, app)); err != nil {
				return err
			}
			err = exec.Run("cp", "-a", filepath.Join(c.Build.Output, c.Name+".app"), "/Applications")
			if err != nil {
				return err
			}
			if c.AppGoSymlink {
				if err := goSymlinkApp(c, app); err != nil {
					return err
				}
			}
			if err := signApp(c, app); err != nil {
				return err
			}
			if len(c.Opens) == 0 {
				return nil
			}
			// so that the system knows the types of files that it opens now
			return exec.Run(lsregister, "-f", app)
		default:
			if err := generate.ContentCode(".", &c.ContentCode); err != nil {
				logx.PrintlnWarn("warning:", err)
			}
			return exec.Major().SetEnv("GOOS", p.OS).SetEnv("GOARCH", runtime.GOARCH).Run("go", "install")
		}
	}
	return nil
}

// appExecutable returns the path of the executable of the given macOS app
// of the given config, as packed by [Pack].
func appExecutable(c *config.Config, app string) string {
	return filepath.Join(app, "Contents", "MacOS", c.Name+".app")
}

// removeExecutable removes the given executable of an installed app, if
// it exists, before a new one is copied there: cp writes over a file in
// place, which macOS then kills when it starts, with an invalid code
// signature, as it keeps the signature of the file that was there.
func removeExecutable(file string) error {
	if err := os.Remove(file); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// goSymlinkApp installs the app of the given config with go install, and
// makes the executable of the given macOS app run the one that it installs
// (see [config.Config.AppGoSymlink]): the executable is a script that
// copies it into the app, next to the script, and signs it and the app,
// when it changes, and runs the copy. The executable that runs must be a
// file in the app (not a symbolic link, which macOS runs the target of),
// signed with the ID of the app, for macOS to allow notifications, for
// example. The script is run once here, with CORE_APP_UPDATE set, which
// only updates the copy.
func goSymlinkApp(c *config.Config, app string) error {
	if err := exec.Major().Run("go", "install"); err != nil {
		return err
	}
	target, err := exec.Output("go", "list", "-f", "{{.Target}}")
	if err != nil {
		return err
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("AppGoSymlink: go install does not install an executable here")
	}
	exe := appExecutable(c, app)
	cp := filepath.Join(filepath.Dir(exe), filepath.Base(target))
	for _, f := range []string{exe, cp} {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	id := shellQuote(c.ID)
	sign := shellQuote(signIdentity())
	script := `#!/bin/sh
# Runs the app installed by go install (AppGoSymlink), which is copied
# into the app and signed when it changes, as macOS only allows
# notifications for an executable in an app that is signed with its ID.
src=` + shellQuote(target) + `
exe="$(dirname "$0")"/` + shellQuote(filepath.Base(cp)) + `
if [ ! -e "$exe" ] || [ "$src" -nt "$exe" ]; then
	if cp "$src" "$exe.new" && codesign -s ` + sign + ` -f -i ` + id + ` "$exe.new" && mv -f "$exe.new" "$exe"; then
		codesign -s ` + sign + ` -f -i ` + id + ` "$(dirname "$0")/../.."
	else
		rm -f "$exe.new"
	fi
fi
[ -n "$CORE_APP_UPDATE" ] && exit 0
exec "$exe" "$@"
`
	if err := os.WriteFile(exe, []byte(script), 0o755); err != nil {
		return err
	}
	if err := exec.Major().SetEnv("CORE_APP_UPDATE", "1").Run(exe); err != nil {
		return err
	}
	if _, err := os.Stat(cp); err != nil {
		return fmt.Errorf("AppGoSymlink: the executable was not copied into the app: %w", err)
	}
	return nil
}

// signApp signs the given macOS app with [signIdentity], with the ID of
// the given config, as macOS only allows notifications for an app that is
// signed with its ID, which the signature of the Go linker does not have.
func signApp(c *config.Config, app string) error {
	return exec.Major().Run("codesign", "-s", signIdentity(), "-f", "-i", c.ID, app)
}

// signIdentity returns the identity for signing macOS apps that are
// installed: the first Apple Development certificate, or else "-" to sign
// ad hoc. The notifications that an app schedules belong to its
// signature, and an ad hoc one changes with every build, so the scheduled
// notifications of the previous builds can no longer be removed, and are
// all shown, whereas those of a certificate stay the same.
func signIdentity() string {
	out, err := exec.Silent().Output("security", "find-identity", "-v", "-p", "codesigning")
	if err != nil {
		return "-"
	}
	return parseSignIdentity(out)
}

// parseSignIdentity returns the hash of the first Apple Development
// identity in the given output of security find-identity, or "-".
func parseSignIdentity(out string) string {
	for line := range strings.Lines(out) {
		f := strings.Fields(line)
		if len(f) >= 3 && strings.HasSuffix(f[0], ")") && strings.Contains(line, `"Apple Development:`) {
			return f[1]
		}
	}
	return "-"
}

// shellQuote returns the given text quoted for a shell.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
