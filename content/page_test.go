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
		Source:       exampleContent,
		Filename:     "button.md",
		Name:         "Button",
		URL:          "button",
		Title:        "Button",
		Categories:   []string{"Widgets"},
		Authors:      "Bea A. Author<sup>1</sup> and Test Ing Name<sup>2</sup>",
		Affiliations: "<sup>1</sup>University of Somwhere <sup>2</sup>University of Elsewhere",
		Abstract:     "The button is an essential element of any GUI framework, with the capability of triggering actions of any sort. Actions are very important because they allow people to actually do something.",
	}, *pg)
}
