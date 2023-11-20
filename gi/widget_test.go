// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"os"
	"testing"

	"github.com/iancoleman/strcase"
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
		typ := typ
		sc := NewEmptyScene()
		sc.NewChild(typ)
		sc.AssertPixelsOnShow(t, strcase.ToSnake(typ.IDName))
	}
}
