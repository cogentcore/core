// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package elide

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestElide(t *testing.T) {
	s := "string for testing purposes"
	assert.Equal(t, "string…", End(s, 7))
	assert.Equal(t, "str…ses", Middle(s, 7))
	assert.Equal(t, "fives", Middle("fives", 5))

	// characters of more than one byte are not cut (as the · of the
	// title of github.com, which made its tab label invalid and blank)
	title := "GitHub · Change is constant. GitHub keeps you ahead. · GitHub · GitHub"
	assert.Equal(t, "GitHub … GitHub", Middle(title, 15))
	assert.True(t, utf8.ValidString(Middle(title, 16)))
	assert.Equal(t, "GitHub ·…", End(title, 9))
	assert.Equal(t, "naïve", Middle("naïve", 5))
}

func TestElideSmall(t *testing.T) {
	assert.Equal(t, "…", End("abc", 1))
	assert.Equal(t, "…", Middle("abc", 1))
	assert.Equal(t, "a…c", Middle("abcdefc", 3))
	assert.NotPanics(t, func() { End("abc", 0) })
	assert.NotPanics(t, func() { Middle("abc", 0) })
	assert.Equal(t, "", End("abc", 0))
	assert.Equal(t, "", Middle("abc", -1))
	assert.Equal(t, "", End("", 0), "an empty string fits")
}

func TestAppName(t *testing.T) {
	assert.Equal(t, "Cogent Code", AppName("Cogent Code"))
	assert.Equal(t, "Cogent Data Browser", AppName("Cogent Data Browser")[:len("Cogent")]+" Data Browser")
	assert.Equal(t, "Cogent Browser", AppName("Cogent Data Browser"))
	assert.Equal(t, "Supercalifragilistic", AppName("Supercalifragilistic"), "one long word")
	assert.Equal(t, "Long Words", AppName("Long Words"))
}
