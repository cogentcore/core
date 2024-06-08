// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command update updates all of the .parse files beneath
// the current directory by opening and saving them.
package main

import (
	"io/fs"
	"path/filepath"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/parse"
)

func main() {
	errors.Log(filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".parse" {
			return nil
		}
		p := parse.NewParser()
		err = p.OpenJSON(path)
		if err != nil {
			return err
		}
		return p.SaveJSON(path)
	}))
}
