// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package assets provides the embedded 3D assets (objects and textures)
// of the xyz examples.
package assets

import "embed"

//go:embed *.png *.obj *.mtl *.blend
var Content embed.FS
