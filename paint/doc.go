// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package paint is the rendering package for Cogent Core.

The Painter provides the rendering state, styling parameters, and methods for
painting. It accumulates all painting actions in a [render.Render]
list, which should be obtained by a call to the [Painter.RenderDone] method
when  done painting (resets list to start fresh).
Pass this [render.Render] list to one or more [render.Renderers] to actually
generate the resulting output. Renderers are independent of the Painter
and the [render.Render] state is entirely self-contained, so rendering
can be done in a separate goroutine etc.

You must import _ "cogentcore.org/core/paint/renderers" to get the default
renderers if using this outside of core which already does this for you.
This sets the New*Renderer functions to point to default implementations.
*/
package paint
