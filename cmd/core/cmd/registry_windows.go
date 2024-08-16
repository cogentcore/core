// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"strings"

	"cogentcore.org/core/base/logx"
	"golang.org/x/sys/windows/registry"
)

// windowsRegistryAddPath adds the given filepath to the system path in the
// Windows registry. If you are adding multiple paths, put a semicolon between
// them, but do NOT add a leading or trailing semicolon; these will be handled
// automatically.
func windowsRegistryAddPath(path string) error {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `System\CurrentControlSet\Control\Session Manager\Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	s, _, err := k.GetStringValue("Path")
	if err != nil {
		return err
	}
	scpath := ";" + path + ";"
	if strings.Contains(s, scpath) {
		fmt.Printf("Path %s already in existing Path: %s\n", path, s)
		return nil
	}
	s += path + ";"
	err = k.SetStringValue("Path", s)
	if err != nil {
		return err
	}
	logx.PrintfDebug("Path %q added to system PATH\n", path)
	return nil
}
