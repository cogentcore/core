// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package main

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// testN is a large number of items, requiring 256MB of storage buffer,
// which exceeds the default WebGPU limit of 128MB, so it tests
// that the device is requesting the full limits of the adapter.
const testN = 16_000_000

func TestComputeNative(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping GPU tests in short mode")
	}
	nerr := compute(testN)
	if nerr != 0 {
		t.Errorf("%d items out of %d do not match", nerr, testN)
	}
}

// resultRegex matches the summary line printed by main.
var resultRegex = regexp.MustCompile(`compute: n: (\d+) errors: (\d+)`)

// TestComputeWeb builds the example for the web using the core tool,
// serves it on localhost, and runs it in headless Chrome, checking
// the console output for the result and any errors.
// Set the CHROME environment variable to specify the Chrome executable.
func TestComputeWeb(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping web test in short mode")
	}
	chrome := findChrome()
	if chrome == "" {
		t.Skip("Chrome not found: set CHROME environment variable to run web test")
	}

	// use the core tool from this module so that it matches the current code.
	build := exec.Command("go", "run", "cogentcore.org/core", "build", "web", "-no-generate-html")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("core build web failed: %v\n%s", err, out)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.FileServer(http.Dir(filepath.Join("bin", "web")))}
	go srv.Serve(ln)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, chrome, "--headless=new", "--enable-unsafe-webgpu",
		"--enable-logging=stderr", "--v=0", "--no-first-run", "--no-default-browser-check",
		"--user-data-dir="+t.TempDir(), "http://"+ln.Addr().String()+"/")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	// Chrome does not exit on its own, so we stop when we see the result.
	var console []string
	found := false
	sc := bufio.NewScanner(stderr)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, ":CONSOLE") {
			continue
		}
		console = append(console, line)
		t.Log(line) // only shown with -v or on failure
		m := resultRegex.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		found = true
		n, _ := strconv.Atoi(m[1])
		nerr, _ := strconv.Atoi(m[2])
		if n != testN {
			t.Errorf("web n: %d != expected: %d", n, testN)
		}
		if nerr != 0 {
			t.Errorf("web: %d items out of %d do not match", nerr, n)
		}
		break
	}
	if !found {
		t.Errorf("did not find result in web console output (timeout: %v)", ctx.Err())
		return
	}
	for _, line := range console {
		// console.error (including Go stderr) is logged as ERROR:CONSOLE, and
		// WebGPU validation errors are logged as warnings mentioning Invalid objects.
		if strings.Contains(line, "ERROR:CONSOLE") || strings.Contains(line, "Uncaught") || strings.Contains(line, "Invalid") {
			t.Errorf("web console error: %s", line)
		}
	}
}

// findChrome returns the path to a Chrome or Chromium executable,
// or "" if not found.
func findChrome() string {
	if c := os.Getenv("CHROME"); c != "" {
		return c
	}
	var paths []string
	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	case "windows":
		paths = []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}
	default:
		for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
			if p, err := exec.LookPath(name); err == nil {
				return p
			}
		}
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
