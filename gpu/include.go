// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"io/fs"
	"log/slog"
	"path"
	"slices"
	"strings"

	"cogentcore.org/core/base/stringsx"
)

// IncludeFS processes #include "file" statements in
// the given code string, using the given file system
// and default path to locate the included files.
func IncludeFS(fsys fs.FS, fpath, code string) string {
	included := map[string]struct{}{}
	return includeFS(fsys, fpath, code, included)
}

func includeFS(fsys fs.FS, fpath, code string, included map[string]struct{}) string {
	fl := stringsx.SplitLines(code)
	nl := len(fl)
	for li := range nl {
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
		fp := path.Join(fpath, fname)
		if _, ok := included[fname]; ok {
			fl[li] = "// " + ln
			continue
		}
		if _, ok := included[fp]; ok {
			fl[li] = "// " + ln
			continue
		}
		inc := fname
		b, err := fs.ReadFile(fsys, fname)
		if err != nil {
			b, err = fs.ReadFile(fsys, fp)
			if err != nil {
				slog.Error("IncludeFS: could not find include", "file", fname, "fpath", fpath)
				continue
			}
			inc = fp
		}
		ins := includeFS(fsys, fpath, string(b), included)
		ol := stringsx.SplitLines(ins)
		fl[li] = "// " + ln
		nl += len(ol)
		fl = slices.Insert(fl, li+1, ol...)
		li += len(ol)
		included[inc] = struct{}{}
	}
	return strings.Join(fl, "\n")
}
