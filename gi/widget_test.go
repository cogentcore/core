// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/iancoleman/strcase"
	"goki.dev/girl/states"
	"goki.dev/icons"
	"goki.dev/laser"
)

func TestMain(m *testing.M) {
	RunTest(func() {
		os.Exit(m.Run())
	})
}

var (
	testStrings = []string{"", "test", "Hello, world!", "123.456", "This is a really long test sentence with a lot of words in it."}
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
