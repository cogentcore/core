// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contentcode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/content"
)

func TestCodeKey(t *testing.T) {
	for _, src := range []string{"", "core.NewButton(b)\n", "\tx := 1\n  y := \"é ü\"\n", "func main() {\n}\n"} {
		if got, want := codeKey(src), content.CodeKey(src); got != want {
			t.Errorf("codeKey(%q) = %q, content.CodeKey = %q", src, got, want)
		}
	}
	if codeKey("a b\n\tc") != codeKey("abc") {
		t.Error("whitespace changes the key")
	}
}

func TestGenerate(t *testing.T) {
	dir, err := os.MkdirTemp("testdata", "tmp-app")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	if err := os.CopyFS(dir, os.DirFS("testdata/app")); err != nil {
		t.Fatal(err)
	}

	// a content app, compiled, with the block that does not type check left out
	c, err := App(dir)
	if err != nil || c == nil || c.Content != "content" {
		t.Fatalf("App = %+v, %v; want the content directory", c, err)
	}
	if ok, err := Generate(c); !ok || err != nil {
		t.Fatalf("Generate = %v, %v", ok, err)
	}
	out, err := os.ReadFile(filepath.Join(dir, c.Output))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"contentCode0, // content/page.md #1", "contentCode1, // content/page.md #2", "_ = x", "content/page.md #3: cannot use 1"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output does not contain %q:\n%s", want, out)
		}
	}

	// up to date, also when only the text around the code changes
	page := filepath.Join(dir, "content", "page.md")
	md, _ := os.ReadFile(page)
	os.WriteFile(page, append([]byte("More text.\n\n"), md...), 0666)
	if ok, err := Generate(c); ok || err != nil {
		t.Errorf("Generate again = %v, %v; want up to date", ok, err)
	}

	// no code blocks: the output is removed
	os.WriteFile(page, []byte("# Page\n"), 0666)
	if ok, err := Generate(c); !ok || err != nil {
		t.Errorf("Generate without code = %v, %v", ok, err)
	}
	if _, err := os.Stat(filepath.Join(dir, c.Output)); err == nil {
		t.Error("output not removed")
	}

	// a go:generate directive gives the arguments
	os.WriteFile(filepath.Join(dir, "gen.go"), []byte("package main\n\n//go:generate go run cogentcore.org/core/content/cmd/contentcode -o other_gen.go\n"), 0666)
	if c, err := App(dir); err != nil || c == nil || c.Output != "other_gen.go" || c.Content != "content" {
		t.Errorf("App with a directive = %+v, %v", c, err)
	}

	// not a content app
	if c, err := App("."); c != nil || err != nil {
		t.Errorf("App(.) = %+v, %v; want nil", c, err)
	}
}
