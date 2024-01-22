// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generate

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"cogentcore.org/core/core/config"
)

var (
	coreExampleStart = []byte("<core-example>")
	coreExampleEnd   = []byte("</core-example>")
	codeStart        = []byte("```go")
	codeEnd          = []byte("```")
	newline          = []byte{'\n'}
)

// Webcore does any necessary generation for webcore.
func Webcore(c *config.Config) error {
	if c.Webcore == "" {
		return nil
	}
	var examples [][]byte
	err := filepath.WalkDir(c.Webcore, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		var curExample [][]byte
		inExample := false
		inCode := false
		for sc.Scan() {
			b := sc.Bytes()

			hasTag := bytes.Contains(b, coreExampleStart)
			if !inExample {
				if !hasTag {
					continue
				}
				inExample = true
				continue
			}
			if hasTag {
				return fmt.Errorf("got two <core-example> tags without a closing tag in %q", path)
			}

			hasCodeStart := bytes.Contains(b, codeStart)
			hasExampleEnd := bytes.Contains(b, coreExampleEnd)
			if !inCode && !hasExampleEnd {
				if !hasCodeStart {
					continue
				}
				inCode = true
				continue
			}

			if bytes.Contains(b, codeEnd) {
				inCode = false
				continue
			}

			if hasExampleEnd {
				examples = append(examples, bytes.Join(curExample, newline))
				curExample = nil
				inExample = false
				continue
			}

			curExample = append(curExample, b)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, e := range examples {
		fmt.Println(string(e))
	}
	return nil
}
