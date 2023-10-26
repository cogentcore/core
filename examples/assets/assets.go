// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package assets

import "embed"

//go:embed *.png *.obj *.mtl *.blend
var Content embed.FS
