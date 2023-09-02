// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"goki.dev/ki/v2/kit"

	vk "github.com/goki/vulkan"
)

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
	TopologiesN                = PatchList + 1
)

//go:generate stringer -type=Topologies

var KiT_Topologies = kit.Enums.AddEnum(TopologiesN, kit.NotBitFlag, nil)
