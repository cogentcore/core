// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"testing"

	"cogentcore.org/core/content/bcontent"
	"github.com/stretchr/testify/assert"
)

func TestNewPage(t *testing.T) {
	pg, err := bcontent.NewPage(exampleContent, "button.md")
	assert.NoError(t, err)
	assert.Equal(t, bcontent.Page{
		Source:     exampleContent,
		Filename:   "button.md",
		Name:       "Button",
		URL:        "button",
		Title:      "Button",
		Categories: []string{"Widgets"},
	}, *pg)
}
