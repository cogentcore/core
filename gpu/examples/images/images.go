// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package images provides the embedded images of the GPU examples.
package images

import "embed"

//go:embed *.png *.jpg
var Images embed.FS
