// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExec(t *testing.T) {
	assert.Equal(t, "README.md\ncmd\nexec.go\nexec_test.go\ninterpreter\nshell.go\ntoken.go\ntranspile.go\ntranspile_test.go", NewShell().Output("ls"))
}
