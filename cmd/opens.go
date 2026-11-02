// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"slices"
	"strings"

	"cogentcore.org/core/base/fileinfo"
)

// openTypes returns the types of files of the given [config.Config.Opens]:
// categories of files or known types of files, as in [fileinfo].
func openTypes(opens []string) ([]fileinfo.MimeType, error) {
	var cats []fileinfo.Categories
	var knowns []fileinfo.Known
	for _, op := range opens {
		var cat fileinfo.Categories
		if cat.SetString(op) == nil && cat != fileinfo.UnknownCategory {
			cats = append(cats, cat)
			continue
		}
		kn, err := fileinfo.KnownByName(op)
		if err != nil || kn == fileinfo.Unknown {
			return nil, fmt.Errorf("Opens: %q is not a category or a known type of file", op)
		}
		knowns = append(knowns, kn)
	}
	var mts []fileinfo.MimeType
	for _, mt := range fileinfo.StandardMimes {
		if len(mt.Exts) == 0 || !(slices.Contains(cats, mt.Cat) || (mt.Known != fileinfo.Unknown && slices.Contains(knowns, mt.Known))) {
			continue
		}
		if slices.ContainsFunc(mts, func(m fileinfo.MimeType) bool { return m.Mime == mt.Mime }) {
			continue
		}
		mts = append(mts, mt)
	}
	return mts, nil
}

// typeName returns the name of the given type of files.
func typeName(mt fileinfo.MimeType) string {
	if mt.Known != fileinfo.Unknown {
		return mt.Known.String()
	}
	return mt.Mime
}

// macTypes are the identifiers of the types of files (UTIs) that macOS has
// for the MIME types, for which the app declares that it opens those types,
// as the other types are matched by the extensions of their files.
var macTypes = map[string]string{
	"text/plain":                "public.plain-text",
	"text/markdown":             "net.daringfireball.markdown",
	"text/html":                 "public.html",
	"text/css":                  "public.css",
	"text/csv":                  "public.comma-separated-values-text",
	"text/tab-separated-values": "public.tab-separated-values-text",
	"application/json":          "public.json",
	"application/xml":           "public.xml",
	"application/yaml":          "public.yaml",
	"application/javascript":    "com.netscape.javascript-source",
	"text/x-csrc":               "public.c-source",
	"text/x-sh":                 "public.shell-script",
	"text/x-csh":                "public.csh-script",
	"text/x-python":             "public.python-script",
	"text/x-perl":               "public.perl-script",
	"text/x-php":                "public.php-script",
	"text/x-ruby":               "public.ruby-script",
	"text/x-java":               "com.sun.java-source",
	"text/x-diff":               "public.patch-file",
}

// macDocType is a type of files that a macOS app opens, in its
// CFBundleDocumentTypes: with the identifier of its type, if macOS has
// one, or else the extensions of its files.
type macDocType struct {
	Name       string
	UTI        string
	Extensions []string
}

// macDocTypes returns the types of files for the CFBundleDocumentTypes
// of a macOS app that opens the given types.
func macDocTypes(mts []fileinfo.MimeType) []macDocType {
	var dts []macDocType
	for _, mt := range mts {
		dt := macDocType{Name: typeName(mt), UTI: macTypes[mt.Mime]}
		if dt.UTI == "" {
			for _, ext := range mt.Exts {
				ext = strings.TrimPrefix(ext, ".")
				if !slices.Contains(dt.Extensions, strings.ToLower(ext)) {
					dt.Extensions = append(dt.Extensions, strings.ToLower(ext))
				}
			}
		}
		dts = append(dts, dt)
	}
	return dts
}

// mimeTypeList returns the MimeType value of a Linux desktop entry
// that opens the given types.
func mimeTypeList(mts []fileinfo.MimeType) string {
	var b strings.Builder
	for _, mt := range mts {
		b.WriteString(mt.Mime + ";")
	}
	return b.String()
}
