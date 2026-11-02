// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (linux && !android) || freebsd || openbsd || netbsd || dragonfly

package apps

import (
	"errors"
	"os/exec"
	"strings"

	"cogentcore.org/core/base/fileinfo"
)

// mimeType returns the MIME type of the given file, as the desktop knows
// it (xdg-mime), or else as [fileinfo] does.
func mimeType(file string) (string, error) {
	if out, err := exec.Command("xdg-mime", "query", "filetype", file).Output(); err == nil {
		if mt, _, _ := strings.Cut(strings.TrimSpace(string(out)), ";"); mt != "" {
			return mt, nil
		}
	}
	mt, _, err := fileinfo.MimeFromFile(file)
	return mt, err
}

func forFile(file string) ([]App, error) {
	mt, err := mimeType(file)
	if err != nil {
		return nil, err
	}
	return xdgApps(mt), nil
}

func openWith(file string, app App) error {
	de := desktopEntries()[app.ID]
	if de == nil {
		return errors.New("apps: no desktop entry for " + app.ID)
	}
	args, err := execArgs(de.Exec, file)
	if err != nil {
		return err
	}
	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

func setDefault(file string, app App) error {
	mt, err := mimeType(file)
	if err != nil {
		return err
	}
	return setXDGDefault(mt, app.ID)
}
