// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"testing"

	"github.com/iancoleman/strcase"
	"goki.dev/goosi"
	"goki.dev/gti"
)

func TestBasicWidgets(t *testing.T) {
	RunTest(func() {
		types := gti.AllEmbeddersOf(WidgetBaseType)
		for _, typ := range types {
			sc := NewScene()
			sc.NewChild(typ)
			NewWindow(sc).Run()
			goosi.AssertCaptureIs(t, strcase.ToSnake(typ.IDName))
		}
	})
}
