// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dirs

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoSrcDir(t *testing.T) {
	dir1 := "fmt"
	expected1 := filepath.Join(build.Default.GOROOT, "src", dir1)
	absDir1, err := GoSrcDir(dir1)
	assert.NoError(t, err)
	assert.Equal(t, expected1, absDir1)

	dir2 := "nonexistent"
	_, err = GoSrcDir(dir2)
	assert.Error(t, err)
	assert.EqualError(t, err, fmt.Sprintf("dirs.GoSrcDir: unable to locate directory (%q) in GOPATH/src/ (%q) or GOROOT/src/pkg/ (%q)", dir2, os.Getenv("GOPATH"), os.Getenv("GOROOT")))
}
