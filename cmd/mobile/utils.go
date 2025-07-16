// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mobile

import (
	"os"
	"runtime"
	"strings"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/cmd/config"
	"golang.org/x/tools/go/packages"
)

var (
	goos   = runtime.GOOS
	goarch = runtime.GOARCH
)

func packagesConfig(t *config.Platform) *packages.Config {
	config := &packages.Config{}
	// Add CGO_ENABLED=1 explicitly since Cgo is disabled when GOOS is different from host OS.
	config.Env = append(os.Environ(), "GOARCH="+t.Arch, "GOOS="+platformOS(t.OS), "CGO_ENABLED=1")
	tags := platformTags(t.OS)

	if len(tags) > 0 {
		config.BuildFlags = []string{"-tags=" + strings.Join(tags, ",")}
	}
	return config
}

func goEnv(name string) string {
	if val := os.Getenv(name); val != "" {
		return val
	}
	val, err := exec.Minor().Output("go", "env", name)
	if err != nil {
		panic(err) // the Go tool was tested to work earlier
	}
	return strings.TrimSpace(string(val))
}
