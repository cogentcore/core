// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestLinkerFlagsID tests that the linker flags set the ID of the app,
// as for its data directory on Android before main.
func TestLinkerFlagsID(t *testing.T) {
	c := &Config{ID: "com.cogent.lists"}
	if f := LinkerFlags(c); !strings.Contains(f, "-X 'cogentcore.org/core/system.AppID=com.cogent.lists'") {
		t.Errorf("no AppID in %q", f)
	}
}

// TestLinkerX tests that the -X linker flags of values with quotes in them
// (as in "O'Reilly", and in SVG icons) are split right by the go command,
// by building a program with them and running it to print the values.
func TestLinkerX(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a program")
	}
	dir := t.TempDir()
	write := func(name, s string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0666); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module x\n\ngo 1.22\n")
	write("main.go", "package main\n\nvar A, B, C, D string\n\nfunc main() { println(A); println(B); println(C); println(D) }\n")
	values := []struct{ value, replace, want string }{
		{"plain text", "", "plain text"},
		{"the website of Randall C. O'Reilly.", "’", "the website of Randall C. O'Reilly."},
		{`<svg width="10"><text font-family="Bob's">hi</text></svg>`, "&#39;", `<svg width="10"><text font-family="Bob&#39;s">hi</text></svg>`},
		{"say \"it's\"", "’", "say \"it’s\""},
	}
	ldflags := "-s -w "
	for i, v := range values {
		ldflags += linkerX("main."+string(rune('A'+i)), v.value, v.replace)
	}
	exe := filepath.Join(dir, "x")
	build := exec.Command("go", "build", "-ldflags", ldflags, "-o", exe, ".")
	build.Dir = dir
	build.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	out, err := exec.Command(exe).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	for i, v := range values {
		if i >= len(got) || got[i] != v.want {
			t.Errorf("value %d: got %q, want %q", i, got, v.want)
		}
	}
}
