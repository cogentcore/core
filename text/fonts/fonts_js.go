// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package fonts

import "embed"

// todo: figure out a more general policy here

// arialjs/*.ttf sfjs/*.ttf notojs/*.ttf

//go:embed robotojs/*.ttf
var DefaultFonts embed.FS
