// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package base

import (
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
)

type ComposerDrawer struct {
	composer.ComposerBase

	Drawer system.Drawer
}

func (cp *ComposerDrawer) Compose() {
	cp.Drawer.Start()
	for _, s := range cp.Sources {
		s.Draw(cp)
	}
	cp.Drawer.End()
}
