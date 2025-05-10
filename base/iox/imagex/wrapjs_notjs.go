// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package imagex

import "image"

// WrapJS returns a JavaScript optimized wrapper around the given
// [image.Image] on web, and just returns the image otherwise.
func WrapJS(src image.Image) image.Image {
	return src
}
