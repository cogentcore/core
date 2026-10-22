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

func TestMarkdownPos(t *testing.T) {
	imports := map[string]string{"core": "cogentcore.org/core/core"}
	// outLine returns the line in the output file and the text of the line
	// of the generated code of b that contains s, with b starting at line 100.
	outLine := func(b *block, s string) (int, string) {
		b.start = 100
		for i, l := range strings.Split(b.code, "\n") {
			if strings.Contains(l, s) {
				return b.start + 1 + i, l
			}
		}
		t.Fatalf("%q not in the code:\n%s", s, b.code)
		return 0, ""
	}

	// statements
	b := &block{line: 10, name: "contentCode3", src: "x := 5\ncore.NewText(b).SetText(undefinedThing)\n\n\n\ny := \"a\"+1\n"}
	if err := b.generate(imports); err != nil {
		t.Fatal(err)
	}
	line, text := outLine(b, "undefinedThing")
	if l, c := b.markdownPos(line, strings.Index(text, "undefinedThing")+1, text); l != 11 || c != 25 {
		t.Errorf("markdownPos = %d:%d; want 11:25", l, c)
	}
	// after empty lines, which the generated code has fewer of; the column
	// is not known, since the code is formatted differently
	line, text = outLine(b, `"a" + 1`)
	if l, c := b.markdownPos(line, strings.Index(text, `"a"`)+1, text); l != 15 || c != 0 {
		t.Errorf("markdownPos = %d:%d; want 15:0", l, c)
	}
	if l, c := b.markdownPos(line, 1, "unknown"); l != 10 || c != 0 {
		t.Errorf("markdownPos of unknown text = %d:%d; want 10:0", l, c)
	}

	// a program, indented in the markdown file
	b = &block{line: 20, indent: 4, name: "contentCode4", src: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(b)\n\tvar s string = 5\n\t_ = s\n}\n"}
	if err := b.generate(imports); err != nil {
		t.Fatal(err)
	}
	line, text = outLine(b, "var s")
	if l, c := b.markdownPos(line, strings.Index(text, "5")+1, text); l != 26 || c != 21 {
		t.Errorf("markdownPos in a program = %d:%d; want 26:21", l, c)
	}
	line, text = outLine(b, "Fprintln")
	if l, _ := b.markdownPos(line, 1, text); l != 25 {
		t.Errorf("markdownPos of a changed line = %d; want 25", l)
	}

	// a syntax error, which is at the end of the code
	b = &block{line: 30, name: "contentCode5", src: "x := 1\ncore.NewButton(b).SetText(\n"}
	if err := b.generate(imports); err == nil || len(b.errs) == 0 {
		t.Fatalf("generate = %v, %v; want errors", err, b.errs)
	}
	for _, e := range b.errs {
		if e.line != 31 || e.col != 0 {
			t.Errorf("syntax error at %d:%d; want 31:0", e.line, e.col)
		}
	}
}

func TestGenerateAppError(t *testing.T) {
	dir, err := os.MkdirTemp("testdata", "tmp-app")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	if err := os.CopyFS(dir, os.DirFS("testdata/app")); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "bad.go"), []byte("package main\n\nfunc bad() int { return \"x\" }\n\nfunc bad2() { undefinedFunc() }\n"), 0666)
	c, err := App(dir)
	if err != nil || c == nil {
		t.Fatalf("App = %+v, %v", c, err)
	}
	// the errors of the app are all listed, with their files
	_, err = Generate(c)
	if err == nil {
		t.Fatal("Generate succeeded with errors in the app")
	}
	bad := filepath.Join(dir, "bad.go")
	for _, want := range []string{"fix these errors first", bad + ":3:25:", bad + ":5:15: undefined: undefinedFunc"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not contain %q: %v", want, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, c.Output)); err == nil {
		t.Error("output written")
	}
}
