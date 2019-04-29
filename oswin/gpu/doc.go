// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package gpu provides an abstract interface to a graphical processing unit (GPU).

Currently it is only supporting OpenGL (version 3.3), but the design should be sufficiently
general to accommodate Vulkan with not too many changes.  Those are the primary
cross-platform GPU systems.

The gpu interfaces provide a chain of construction, starting with the GPU which can
create a Program or a Pipeline (collection of programs), which is the core of the
system, defining different Shader programs that run on the GPU.

Each Program has Uniform variables (constants across all GPU cores) and Input
Vectors which are the vectorized data that each GPU core computes in parallel.
For graphics, the Vectors are verticies, normals, etc.
The Output vectors transfer information from one shader program to another
(e.g., the Vertex shader sends output to the Fragment shader).
Programs should be written in 3.3 GLSL but DO NOT include the version info
as it will be automatically added.

All Vectors processed by a Program must be contained in a SINGLE VectorsBuffer
which can interleave or just append the data from multiple Vectors into a single
continguous chunk of memory.  Typically it is more efficient to use an
indexed view onto the Vectors data, provided by the IndexesBuffer.

The BufferMgr manages a VectorsBuffer and IndexesBuffer, and corresponds to the
Vertex Array Object in OpenGL.

For typical usage (e.g., in gi3d), there is a different Program for each different
type of Material, and e.g., the Uniform's define the camera's view transform
and any uniform colors, etc.  Each Object or Shape (or Geometry) has a BufferMgr
configured with the vertex, normal, etc data for its particular shape.
*/
package gpu
