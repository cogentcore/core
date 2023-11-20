// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iancoleman/strcase"
	"goki.dev/grr"
	"goki.dev/gti"
)

func TestMain(m *testing.M) {
	RunTest(func() {
		os.Exit(m.Run())
	})
}

func TestBasicWidgets(t *testing.T) {
	types := gti.AllEmbeddersOf(WidgetBaseType)
	for _, typ := range types {
		sc := NewEmptyScene()
		sc.NewChild(typ)
		sc.AssertPixelsOnShow(t, strcase.ToSnake(typ.IDName))
	}
}

func TestTextWidgets(t *testing.T) {
	strs := []string{"test", "Hello, world!", "123.456", "This is a really long test sentence with a lot of words in it."}

	funcs := map[string]func(par Widget, text string){
		"button": func(par Widget, text string) {
			NewButton(par).SetText(text)
		},
		"label": func(par Widget, text string) {
			NewLabel(par).SetText(text)
		},
		"switch": func(par Widget, text string) {
			NewSwitch(par).SetText(text)
		},
		"tab": func(par Widget, text string) {
			NewTab(par).SetText(text)
		},
	}
	for nm, f := range funcs {
		for _, str := range strs {
			sc := NewEmptyScene()
			f(sc, str)
			fw, _, _ := strings.Cut(str, " ")
			grr.Log0(os.MkdirAll(filepath.Join("testdata", nm), 0750))
			sc.AssertPixelsOnShow(t, filepath.Join(nm, "text_"+strcase.ToSnake(fw)))
		}
	}
}
