// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package paint is the rendering package for Cogent Core.
The Painter provides the rendering state, styling parameters, and methods for
painting. The [State] contains a list of Renderers that will actually
render the paint commands. For improved performance, and sensible results
with document-style renderers (e.g., SVG, PDF), an entire scene should be
rendered, followed by a RenderDone call that actually performs the rendering
using a list of rendering commands stored in the [State.Render]. Also ensure
that items used in a rendering pass remain valid through the RenderDone step,
and are not reused within a single pass.

You must import _ "cogentcore.org/core/paint/renderers" to get the default
renderers if using this outside of core which already does this for you.
*/
package paint
