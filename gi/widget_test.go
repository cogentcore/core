// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"os"
	"testing"

	"github.com/iancoleman/strcase"
	"goki.dev/goosi"
	"goki.dev/gti"
)

func TestMain(m *testing.M) {
	RunTest(func() {
		os.Exit(m.Run())
	})
}

func TestBasicWidgets(t *testing.T) {
	types := gti.AllEmbeddersOf(WidgetBaseType)
	sc := NewScene()
	NewWindow(sc).Run()
	for _, typ := range types {
		updt := sc.UpdateStart()
		sc.DeleteChildAtIndex(1, true)
		sc.NewChild(typ)
		sc.Update()
		sc.UpdateEndLayout(updt)
		goosi.AssertCaptureIs(t, strcase.ToSnake(typ.IDName))
	}
}
