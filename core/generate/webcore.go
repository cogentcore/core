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
	"cogentcore.org/core/gengo"
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
	examples, err := GetWebcoreExamples(c)
	if err != nil {
		return err
	}
	return WriteWebcoregen(c, examples)
}

// GetWebcoreExamples collects and returns all of the webcore examples.
func GetWebcoreExamples(c *config.Config) ([][]byte, error) {
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
		return nil, err
	}
	return examples, nil
}

// WriteWebcoregen constructs the webcoregen.go file from the given examples.
func WriteWebcoregen(c *config.Config, examples [][]byte) error {
	b := &bytes.Buffer{}
	gengo.PrintHeader(b, "main")
	b.WriteString(`func init() {
	maps.Copy(webcore.Examples, WebcoreExamples)
}

// WebcoreExamples are the compiled webcore examples for this app.
var WebcoreExamples = map[string]func(parent gi.Widget){`)
	for i, example := range examples {
		fmt.Fprintf(b, `
	"%d": func(parent gi.Widget){%s},`, i, example)
	}
	b.WriteString("\n}")

	return gengo.Write("webcoregen.go", b.Bytes(), nil)
}
