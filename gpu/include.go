// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"io/fs"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/stringsx"
)

// IncludeFS processes #include "file" statements in
// the given code string, using the given file system
// and default path to locate the included files.
func IncludeFS(fsys fs.FS, path, code string) string {
	fl := stringsx.SplitLines(code)
	nl := len(fl)
	for li := nl - 1; li >= 0; li-- {
		ln := fl[li]
		if !strings.HasPrefix(ln, `#include "`) {
			continue
		}
		fn := ln[10:]
		qi := strings.Index(fn, `"`)
		if qi < 0 {
			slog.Error("IncludeFS: malformed #include: no final quote")
			continue
		}
		fname := fn[:qi]
		b, err := fs.ReadFile(fsys, fname)
		if err != nil {
			b, err = fs.ReadFile(fsys, filepath.Join(path, fname))
			if err != nil {
				slog.Error("IncludeFS: could not find include", "file", fname, "path", path)
				continue
			}
		}
		ol := stringsx.SplitLines(string(b))
		fl[li] = "// " + ln
		fl = slices.Insert(fl, li+1, ol...)
	}
	return strings.Join(fl, "\n")
}
