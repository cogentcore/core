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
	"regexp"

	"cogentcore.org/core/core/config"
	"cogentcore.org/core/gengo"
	"cogentcore.org/core/ordmap"
)

var (
	coreExampleStart = []byte("<core-example")
	coreExampleEnd   = []byte("</core-example>")
	codeStart        = []byte("```go")
	codeEnd          = []byte("```")
	newline          = []byte{'\n'}

	idRegex        = regexp.MustCompile(`id="(.+)"`)
	idRegexReplace = []byte("$1")
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
func GetWebcoreExamples(c *config.Config) (ordmap.Map[string, []byte], error) {
	var examples ordmap.Map[string, []byte]
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
		exampleID := ""
		for sc.Scan() {
			b := sc.Bytes()

			hasTag := bytes.Contains(b, coreExampleStart)
			if !inExample {
				if !hasTag {
					continue
				}
				inExample = true
				exampleID = string(idRegex.ReplaceAll(b, idRegexReplace))
				fmt.Println(exampleID)
				if exampleID == "" {
					return fmt.Errorf("missing ID for <core-example> tag in %q", path)
				}
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
				examples.Add(exampleID, bytes.Join(curExample, newline))
				curExample = nil
				inExample = false
				continue
			}

			curExample = append(curExample, b)
		}
		return nil
	})
	return examples, err
}

// WriteWebcoregen constructs the webcoregen.go file from the given examples.
func WriteWebcoregen(c *config.Config, examples ordmap.Map[string, []byte]) error {
	b := &bytes.Buffer{}
	gengo.PrintHeader(b, "main")
	b.WriteString(`func init() {
	maps.Copy(webcore.Examples, WebcoreExamples)
}

// WebcoreExamples are the compiled webcore examples for this app.
var WebcoreExamples = map[string]func(parent gi.Widget){`)
	for _, kv := range examples.Order {
		fmt.Fprintf(b, `
	%q: func(parent gi.Widget){%s},`, kv.Key, kv.Val)
	}
	b.WriteString("\n}")

	return gengo.Write("webcoregen.go", b.Bytes(), nil)
}
