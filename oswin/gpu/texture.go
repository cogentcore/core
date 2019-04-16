// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import "github.com/goki/gi/oswin"

// Texture2D manages a 2D texture, including loading from an image file
// and activating on GPU.  The oswin.Texture interface presents the
// subset of the
type Texture2D interface {
	oswin.Texture
}
