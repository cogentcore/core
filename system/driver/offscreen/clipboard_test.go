// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offscreen

import (
	"testing"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"github.com/stretchr/testify/assert"
)

func TestClipboard(t *testing.T) {
	cl := &Clipboard{}
	assert.True(t, cl.IsEmpty())
	assert.Nil(t, cl.Read([]string{mimedata.TextPlain}))

	assert.NoError(t, cl.Write(mimedata.NewText("hello")))
	assert.False(t, cl.IsEmpty())
	assert.Equal(t, "hello", cl.Read([]string{mimedata.TextPlain}).Text(mimedata.TextPlain))
	assert.Nil(t, cl.Read([]string{"image/png"}), "only text is read")

	md := mimedata.NewTextPlus("path", mimedata.DataJson, []byte(`{"a":1}`))
	assert.NoError(t, cl.Write(md))
	got := cl.Read([]string{mimedata.DataJson})
	assert.Len(t, got, 2, "multiple items round trip through multipart")
	assert.Equal(t, `{"a":1}`, string(got.TypeData(mimedata.DataJson)))
	assert.Equal(t, "path", got.Text(mimedata.TextPlain))

	cl.Clear()
	assert.True(t, cl.IsEmpty())
}
