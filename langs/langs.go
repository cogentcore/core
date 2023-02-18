// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package langs

import (
	"embed"
	"path/filepath"
	"strings"

	"github.com/goki/pi/filecat"
)

//go:embed */*.pi
var content embed.FS

func OpenParser(sl filecat.Supported) ([]byte, error) {
	ln := strings.ToLower(sl.String())
	lndir := ln
	if lndir == "go" {
		lndir = "golang" // can't name a package "go"..
	}
	fn := filepath.Join(lndir, ln+".pi")
	return content.ReadFile(fn)
}
