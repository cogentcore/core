// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xmlx

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCharsetReader(t *testing.T) {
	type doc struct {
		Text string `xml:",chardata"`
	}
	for _, enc := range []string{"ISO-8859-1", "windows-1252", "utf-8"} {
		text := "caf\xe9"
		if enc == "utf-8" {
			text = "café"
		}
		decoder := xml.NewDecoder(strings.NewReader(`<?xml version="1.0" encoding="` + enc + `"?><a>` + text + `</a>`))
		decoder.CharsetReader = CharsetReader
		var d doc
		assert.NoError(t, decoder.Decode(&d), enc)
		assert.Equal(t, "café", d.Text, enc)
	}
	_, err := CharsetReader("no-such-charset", strings.NewReader("a"))
	assert.Error(t, err)
}
