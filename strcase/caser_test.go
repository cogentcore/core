// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/ettle/strcase
// Copyright (c) 2020 Liyan David Chang under the MIT License

package strcase

import (
	"reflect"
	"testing"
	"unicode"
)

func TestCaserAll(t *testing.T) {
	c := NewCaser(true, nil, nil)

	type data struct {
		input  string
		snake  string
		SNAKE  string
		kebab  string
		KEBAB  string
		pascal string
		camel  string
		title  string
	}
	for _, test := range []data{
		{
			input:  "Hello world!",
			snake:  "hello_world!",
			SNAKE:  "HELLO_WORLD!",
			kebab:  "hello-world!",
			KEBAB:  "HELLO-WORLD!",
			pascal: "HelloWorld!",
			camel:  "helloWorld!",
			title:  "Hello World!",
		},
	} {
		t.Run(test.input, func(t *testing.T) {
			output := data{
				input:  test.input,
				snake:  c.ToSnake(test.input),
				SNAKE:  c.ToSNAKE(test.input),
				kebab:  c.ToKebab(test.input),
				KEBAB:  c.ToKEBAB(test.input),
				pascal: c.ToPascal(test.input),
				camel:  c.ToCamel(test.input),
				title:  c.ToCase(test.input, TitleCase, ' '),
			}
			assertTrue(t, test == output)
		})
	}
}

func TestNewCaser(t *testing.T) {
	t.Run("Has defaults when unspecified", func(t *testing.T) {
		c := NewCaser(true, nil, nil)
		assertTrue(t, reflect.DeepEqual(golintInitialisms, c.initialisms))
		assertTrue(t, c.splitFn != nil)
	})
	t.Run("Merges", func(t *testing.T) {
		c := NewCaser(true, map[string]bool{"SSL": true, "HTML": false}, nil)
		assertTrue(t, !reflect.DeepEqual(golintInitialisms, c.initialisms))
		assertTrue(t, c.initialisms["UUID"])
		assertTrue(t, c.initialisms["SSL"])
		assertTrue(t, !c.initialisms["HTML"])
		assertTrue(t, c.splitFn != nil)
	})

	t.Run("No Go initialisms", func(t *testing.T) {
		c := NewCaser(false, map[string]bool{"SSL": true, "HTML": false}, NewSplitFn([]rune{' '}))
		assertTrue(t, !reflect.DeepEqual(golintInitialisms, c.initialisms))
		assertTrue(t, !c.initialisms["UUID"])
		assertTrue(t, c.initialisms["SSL"])
		assertTrue(t, !c.initialisms["HTML"])
		assertEqual(t, "hTml with SSL", c.ToCase("hTml with SsL", Original, ' '))
		assertTrue(t, c.splitFn != nil)
	})

	t.Run("Preserve number formatting", func(t *testing.T) {
		c := NewCaser(
			false,
			map[string]bool{"SSL": true, "HTML": false},
			NewSplitFn(
				[]rune{'*', '.', ','},
				SplitCase,
				SplitAcronym,
				PreserveNumberFormatting,
			))
		assertTrue(t, !reflect.DeepEqual(golintInitialisms, c.initialisms))
		assertEqual(t, "http200", c.ToSnake("http200"))
		assertEqual(t, "VERSION2.3_R3_8A_HTTP_ERROR_CODE", c.ToSNAKE("version2.3R3*8a,HTTPErrorCode"))
	})

	t.Run("Preserve number formatting and split before and after number", func(t *testing.T) {
		c := NewCaser(
			false,
			map[string]bool{"SSL": true, "HTML": false},
			NewSplitFn(
				[]rune{'*', '.', ','},
				SplitCase,
				SplitAcronym,
				PreserveNumberFormatting,
				SplitBeforeNumber,
				SplitAfterNumber,
			))
		assertEqual(t, "http_200", c.ToSnake("http200"))
		assertEqual(t, "VERSION_2.3_R_3_8_A_HTTP_ERROR_CODE", c.ToSNAKE("version2.3R3*8a,HTTPErrorCode"))
	})

	t.Run("Skip non letters", func(t *testing.T) {
		c := NewCaser(
			false,
			nil,
			func(prec, curr, next rune) SplitAction {
				if unicode.IsNumber(curr) {
					return Noop
				} else if unicode.IsSpace(curr) {
					return SkipSplit
				}
				return Skip
			})
		assertEqual(t, "", c.ToSnake(""))
		assertEqual(t, "1130_23_2009", c.ToCase("DateTime: 11:30 AM May 23rd, 2009", Original, '_'))
	})
}
