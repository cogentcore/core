// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package svg provides SVG rendering classes, I/O parsing: full SVG rendering

SVG currently supports most of SVG, but not:

	* Flow
	* Filter Effects
	* 3D Perspective transforms

See gi/examples/svg for a basic SVG viewer app, using the svg.Editor, which
will ultimately be expanded to support more advanced editing.  Also in that
directory are a number of test files that stress different aspects of
rendering.

svg.NodeBase is the base type for all SVG elements -- unlike Widget nodes, SVG
nodes do not use layout logic, and just draw directly into a parent SVG
viewport, with cumulative transforms determining drawing position, etc.  The
BBox values are only valid after rendering for these nodes.

It uses srwiley/rasterx for SVG-compatible rasterization, and the gi.Paint
interface for drawing.

The Path element uses a compiled bytecode version of the Data path for
increased speed.

*/
package svg
