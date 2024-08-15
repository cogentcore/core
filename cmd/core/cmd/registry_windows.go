// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"strings"

	"cogentcore.org/core/base/errors"
	"golang.org/x/sys/windows/registry"
)

// WindowsRegistryAddPath adds given path to the system path in the
// windows registry.  Path string must be properly escaped when passed
// as a double-quote string, e.g., "C:\\w64devkit\\bin".
// If adding multiple paths, put ; between them, but do NOT add a leading
// or trailing semicolon -- these will be handled automatically.
func WindowsRegistryAddPath(path string) error {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `System\CurrentControlSet\Control\Session Manager\Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if errors.Log(err) != nil {
		return err
	}
	defer k.Close()

	s, _, err := k.GetStringValue("Path")
	if errors.Log(err) != nil {
		return err
	}
	scpath := ";" + path + ";"
	if strings.Contains(s, scpath) {
		fmt.Printf("Path %s already in existing Path: %s\n", path, s)
		return nil
	}
	s += path + ";"
	err = k.SetStringValue("Path", s)
	if errors.Log(err) != nil {
		return err
	}
	fmt.Printf("Path: %s added to system Path: re-start any open shells to use\n", path)
	return nil
}
