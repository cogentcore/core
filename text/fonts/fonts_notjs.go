// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package fonts

import "embed"

//go:embed noto/*.ttf robotomono/*.ttf
var Default embed.FS
