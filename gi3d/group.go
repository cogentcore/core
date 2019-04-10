// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

// Group collects individual elements in a scene but does not have geometry of
// its own.  It does have a transform that applies to all nodes under it.
type Group struct {
	Node3DBase
}
