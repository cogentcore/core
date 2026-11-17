// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSPackageJSON(t *testing.T) {
	m := &goscriptManifest{Dependencies: map[string]string{"pako": "3.0.2", "@noble/hashes": "2.2.0"}}
	b, err := jsPackageJSON(m)
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"module","private":true,"dependencies":{"@noble/hashes":"2.2.0","pako":"3.0.2"}}`+"\n", string(b))
}

func TestGoscriptPackageJSON(t *testing.T) {
	if ToolPath("goscript") == "" {
		t.Skip("goscript is not installed")
	}
	// The package.json of the installed goscript has the npm dependencies
	// of its runtime, and the tsrs version that goscript test uses.
	m, err := goscriptPackageJSON()
	assert.NoError(t, err)
	assert.Contains(t, m.Dependencies, "@noble/hashes")
	assert.NotEmpty(t, m.DevDependencies["tsrs"])
}
