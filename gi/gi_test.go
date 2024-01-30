// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"cogentcore.org/core/icons"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/states"
	"github.com/iancoleman/strcase"
)

func TestBasic(t *testing.T) {
	b := NewBody()
	NewLabel(b).SetText("Test")
	b.AssertRender(t, "basic")
}

var (
	testStrings = []string{"", "Test", "Hello, world!", "123.456", "This is a really long test sentence with a lot of words in it."}
	testIcons   = []icons.Icon{"", icons.Search}
	testStates  = [][]states.States{
		{},
		{states.Disabled},
		{states.Hovered},
		{states.Focused},
		{states.Active},
		{states.Hovered, states.Focused},
		{states.Hovered, states.Active},
		{states.Focused, states.Active},
		{states.Hovered, states.Focused, states.Active},
	}
)

func testName(elems ...any) string {
	strs := []string{}
	for _, elem := range elems {
		str := laser.ToString(elem)
		fields := strings.FieldsFunc(str, func(r rune) bool {
			return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '|')
		})
		if len(fields) > 0 {
			f := strcase.ToSnake(fields[0])
			f = strings.ReplaceAll(f, "|", "_")
			strs = append(strs, f)
		} else {
			strs = append(strs, "none")
		}
	}
	return filepath.Join(strs...)
}
