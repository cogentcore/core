// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package mimedata

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWebMultipart checks that the web versions of ToMultipart and
// FromMultipart write and read the same format as mime/multipart.
func TestWebMultipart(t *testing.T) {
	mimes := []Mimes{
		NewText("hello"),
		NewTextPlus("some text\nwith lines\r\n", "image/png", []byte{0, 1, 2, 3, 250, 255}),
		{NewTextData("first"), NewTextData(""), &Data{Type: DataJson, Data: []byte(`{"a": 1}`)}, &Data{Type: "application/octet-stream", Data: []byte("--\r\n--x")}},
		{},
	}
	for _, mi := range mimes {
		// The same output except for the random boundary.
		native, web := mi.ToMultipart(), writeMultipart(mi)
		_, _, nb, nbody := IsMultipart(native)
		isMulti, _, wb, wbody := IsMultipart(web)
		assert.True(t, isMulti)
		assert.Equal(t, len(nb), len(wb))
		assert.Equal(t, strings.ReplaceAll(string(native), nb, "B"), strings.ReplaceAll(string(web), wb, "B"))

		// Both readers read both outputs the same.
		want := FromMultipart(nbody, nb)
		assert.Equal(t, want, readMultipart(nbody, nb))
		assert.Equal(t, want, FromMultipart(wbody, wb))
		assert.Equal(t, want, readMultipart(wbody, wb))
		assert.Len(t, want, len(mi))
	}
}
