// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	vk "github.com/goki/vulkan"
)

// todo: this does not parse in enumgen

// Topologies are the different vertex topology
type Topologies int32

const (
	PointList                  = Topologies(vk.PrimitiveTopologyPointList)
	LineList                   = Topologies(vk.PrimitiveTopologyLineList)
	LineStrip                  = Topologies(vk.PrimitiveTopologyLineStrip)
	TriangleList               = Topologies(vk.PrimitiveTopologyTriangleList)
	TriangleStrip              = Topologies(vk.PrimitiveTopologyTriangleStrip)
	TriangleFan                = Topologies(vk.PrimitiveTopologyTriangleFan)
	LineListWithAdjacency      = Topologies(vk.PrimitiveTopologyLineListWithAdjacency)
	LineStripWithAdjacency     = Topologies(vk.PrimitiveTopologyLineStripWithAdjacency)
	TriangleListWithAdjacency  = Topologies(vk.PrimitiveTopologyTriangleListWithAdjacency)
	TriangleStripWithAdjacency = Topologies(vk.PrimitiveTopologyTriangleStripWithAdjacency)
	PatchList                  = Topologies(vk.PrimitiveTopologyPatchList)
)
