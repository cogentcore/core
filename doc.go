// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package gi is the top-level repository for the GoGi GUI framework.

All of the code is in the sub-packages within this repository:

* examples: example main programs for learning how to use GoGi -- see
            widgets and its README file for a basic introduction.  marbles is fun!

* gist: css-based styling settings, including Color

* girl: rendering library, can be used standalone, SVG compliant

* gi: the main 2D GUI Node, Widgets, and Window

* giv: more complex Views of Go data structures, supporting Model-View paradigm.

* svg: full SVG rendering framework, used for Icons in gi.

* gi3d: 3D rendering of a Scene within 2D windows -- full interactive 3D scenegraph.

* histyle: text syntax-based highlighting styles -- used in giv.TextView

* oswin: OS-specific framework for low-level rendering, event interface,
         including GPU abstraction (OpenGL for now, ultimately Vulcan)

* python: access all of GoGi from within Python using GoPy system.

* undo: generic undo / redo functionality using text blobs of state
        compressed using diff, or commands.

*/
package gi
