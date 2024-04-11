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
	"strconv"
	"strings"

	"cogentcore.org/core/cmd/core/config"
	"cogentcore.org/core/generate"
	"cogentcore.org/core/ordmap"
	"cogentcore.org/core/webcore/wpath"
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
		gotNewBody := false
		gotMain := false
		numExamples := 0
		for sc.Scan() {
			b := sc.Bytes()

			if !inExample {
				if bytes.HasPrefix(b, []byte("```Go")) {
					inExample = true
				}
				continue
			}

			if bytes.HasPrefix(b, []byte("func main() {")) {
				gotMain = true
			}

			// core.NewBody in a main function counts as a new start so that full examples work
			if gotMain && !gotNewBody && bytes.Contains(b, []byte("core.NewBody(")) {
				gotNewBody = true
				curExample = nil
				curExample = append(curExample, []byte("b := parent"))
				continue
			}

			// RunMainWindow() counts as a quasi-end so that full examples work
			if string(b) == "```" || bytes.Contains(b, []byte("RunMainWindow()")) {
				if curExample == nil {
					continue
				}
				rel, err := filepath.Rel(c.Webcore, path)
				if err != nil {
					return err
				}
				rel = strings.ReplaceAll(rel, `\`, "/")
				rel = strings.TrimSuffix(rel, filepath.Ext(rel))
				rel = strings.TrimSuffix(rel, "/index")
				rel = wpath.Format(rel)
				id := rel + "-" + strconv.Itoa(numExamples)
				examples.Add(id, bytes.Join(curExample, []byte{'\n'}))
				curExample = nil
				inExample = false
				gotNewBody = false
				numExamples++
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
	generate.PrintHeader(b, "main")
	b.WriteString(`func init() {
	maps.Copy(webcore.Examples, WebcoreExamples)
}

// WebcoreExamples are the compiled webcore examples for this app.
var WebcoreExamples = map[string]func(parent core.Widget){`)
	for _, kv := range examples.Order {
		fmt.Fprintf(b, `
	%q: func(parent core.Widget){%s%s},`, kv.Key, "\n", kv.Value)
	}
	b.WriteString("\n}")

	return generate.Write("webcoregen.go", b.Bytes(), nil)
}
