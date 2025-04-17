// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
)

// GenerateMarkdown extracts markdown citations in the format [@Ref; @Ref]
// from .md markdown files in given directory, looking up in given source [KeyList],
// and writing the results in given style to given .md file (references.md default).
// Heading is written first: must include the appropriate markdown heading level
// (## typically). Returns the [KeyList] of references that were cited.
func GenerateMarkdown(dir, refFile, heading string, kl *KeyList, sty Styles) (*KeyList, error) {
	cited := &KeyList{}
	if dir == "" {
		dir = "./"
	}
	mds := fsx.Filenames(dir, ".md")
	if len(mds) == 0 {
		return cited, errors.New("No .md files found in: " + dir)
	}
	var errs []error
	for i := range mds {
		mds[i] = filepath.Join(dir, mds[i])
	}
	err := ExtractMarkdownCites(mds, kl, cited)
	if err != nil {
		errs = append(errs, err)
	}
	if refFile == "" {
		refFile = filepath.Join(dir, "references.md")
	}
	of, err := os.Create(refFile)
	if err != nil {
		errs = append(errs, err)
		return cited, errors.Join(errs...)
	}
	defer of.Close()
	if heading != "" {
		of.WriteString(heading + "\n\n")
	}
	err = WriteRefsMarkdown(of, cited, sty)
	if err != nil {
		errs = append(errs, err)
	}
	return cited, errors.Join(errs...)
}

// ExtractMarkdownCites extracts markdown citations in the format [@Ref; @Ref]
// from given list of .md files, looking up in given source [KeyList], adding to cited.
func ExtractMarkdownCites(files []string, src, cited *KeyList) error {
	exp := regexp.MustCompile(`\[(@\^?([[:alnum:]]+-?)+(;[[:blank:]]+)?)+\]`)
	var errs []error
	for _, fn := range files {
		f, err := os.Open(fn)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		scan := bufio.NewScanner(f)
		for scan.Scan() {
			cs := exp.FindAllString(string(scan.Bytes()), -1)
			for _, c := range cs {
				tc := c[1 : len(c)-1]
				sp := strings.Split(tc, "@")
				for _, ac := range sp {
					a := strings.TrimSpace(ac)
					a = strings.TrimSuffix(a, ";")
					if a == "" {
						continue
					}
					if a[0] == '^' {
						a = a[1:]
					}
					it, has := src.AtTry(a)
					if !has {
						err = errors.New("citation not found: " + a)
						errs = append(errs, err)
						continue
					}
					cited.Add(a, it)
				}
			}
		}
		f.Close()
	}
	return errors.Join(errs...)
}

// WriteRefsMarkdown writes references from given [KeyList] to a
// markdown file.
func WriteRefsMarkdown(w io.Writer, kl *KeyList, sty Styles) error {
	refs, items := Refs(sty, kl)
	for i, ref := range refs {
		it := items[i]
		_, err := w.Write([]byte(`<p id="` + it.CitationKey + `">`))
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(string(ref.Join()) + "</p>\n\n")) // todo: ref to markdown!!
		if err != nil {
			return err
		}
	}
	return nil
}
