// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// processDir returns the working directory of the process of the given
// id, or "" if it cannot be found: from /proc on Linux, and with lsof on
// macOS and the BSDs.
func processDir(pid int) string {
	switch runtime.GOOS {
	case "linux", "android":
		dir, _ := os.Readlink("/proc/" + strconv.Itoa(pid) + "/cwd")
		return dir
	case "darwin", "freebsd", "netbsd", "openbsd", "dragonfly":
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		lsof, err := exec.LookPath("lsof")
		if err != nil {
			lsof = "/usr/sbin/lsof" // (often not in the path)
		}
		out, err := exec.CommandContext(ctx, lsof, "-a", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn").Output()
		if err != nil {
			return ""
		}
		for line := range strings.Lines(string(out)) {
			if dir, ok := strings.CutPrefix(strings.TrimRight(line, "\n"), "n"); ok {
				return dir
			}
		}
	}
	return ""
}
