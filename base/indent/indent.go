// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package indent provides indentation generation methods.
package indent

//go:generate core generate

import (
	"bytes"
	"strings"
)

// Character is the type of indentation character to use.
type Character int32 //enums:enum

const (
	// Tab indicates to use tabs for indentation.
	Tab Character = iota

	// Space indicates to use spaces for indentation.
	Space
)

// Tabs returns a string of n tabs.
func Tabs(n int) string {
	return strings.Repeat("\t", n)
}

// TabBytes returns []byte of n tabs.
func TabBytes(n int) []byte {
	return bytes.Repeat([]byte("\t"), n)
}

// Spaces returns a string of n*width spaces.
func Spaces(n, width int) string {
	return strings.Repeat(" ", n*width)
}

// SpaceBytes returns a []byte of n*width spaces.
func SpaceBytes(n, width int) []byte {
	return bytes.Repeat([]byte(" "), n*width)
}

// String returns a string of n tabs or n*width spaces depending on the indent character.
func String(ich Character, n, width int) string {
	if ich == Tab {
		return Tabs(n)
	}
	return Spaces(n, width)
}

// Bytes returns []byte of n tabs or n*width spaces depending on the indent character.
func Bytes(ich Character, n, width int) []byte {
	if ich == Tab {
		return TabBytes(n)
	}
	return SpaceBytes(n, width)
}

// Len returns the length of the indent string given indent character and indent level.
func Len(ich Character, n, width int) int {
	if ich == Tab {
		return n
	}
	return n * width
}
