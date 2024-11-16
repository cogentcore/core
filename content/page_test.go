// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPage(t *testing.T) {
	pg, err := NewPage(exampleContent, "button.md")
	assert.NoError(t, err)
	assert.Equal(t, Page{
		Source:     exampleContent,
		Filename:   "button.md",
		Name:       "Button",
		URL:        "Button",
		Categories: []string{"Widgets"},
	}, *pg)
}