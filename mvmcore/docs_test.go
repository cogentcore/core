// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mvmcore

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"cogentcore.org/core/core"
)

// codeBlock is a runnable Go code block of a markdown file.
type codeBlock struct {
	file string
	line int
	src  string
}

var (
	fenceStart = regexp.MustCompile("^(\\s*)```Go\\s*$")
	fenceEnd   = regexp.MustCompile("^\\s*```\\s*$")
)

// docsBlocks returns the runnable Go code blocks of the markdown files of
// the docs.
func docsBlocks(t *testing.T) []codeBlock {
	files, err := filepath.Glob("../docs/content/*.md")
	if err != nil {
		t.Fatal(err)
	}
	var blocks []codeBlock
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(data), "\n")
		for i := 0; i < len(lines); i++ {
			m := fenceStart.FindStringSubmatch(lines[i])
			if m == nil {
				continue
			}
			start := i + 1
			var src []string
			for i++; i < len(lines) && !fenceEnd.MatchString(lines[i]); i++ {
				src = append(src, strings.TrimPrefix(lines[i], m[1]))
			}
			blocks = append(blocks, codeBlock{filepath.Base(f), start, strings.Join(src, "\n")})
		}
	}
	return blocks
}

// codeSymbols are the names that the docs add to content.CodeSymbols.
var codeSymbols = regexp.MustCompile(`\b(econtent|myImage|mySVG|myFile)\b`)

// TestDocs runs the runnable Go code blocks of the docs, other than those
// that use the embedded files of the docs app.
func TestDocs(t *testing.T) {
	blocks := docsBlocks(t)
	failed := 0
	for _, bl := range blocks {
		if codeSymbols.MatchString(bl.src) {
			continue
		}
		if err := runBlock(bl.src); err != nil {
			failed++
			t.Errorf("%s:%d: %v", bl.file, bl.line, firstLine(err))
		}
	}
	t.Logf("%d of %d code blocks failed", failed, len(blocks))
}

// runBlock runs src in a new interpreter, in a new frame.
func runBlock(src string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("host panic: %v", r)
		}
	}()
	var out bytes.Buffer
	in, err := NewGo(&out)
	if err != nil {
		return err
	}
	parent := core.NewFrame(core.NewBody())
	core.ExternalParent = parent
	*interpreterParent = parent
	_, err = in.Eval(src)
	return err
}

// firstLine returns the first line of the message of err.
func firstLine(err error) string {
	s := err.Error()
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
