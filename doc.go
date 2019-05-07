// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package pi provides the top-level repository for the GoPi interactive parser system.

The code is organized into the various sub-packages, dealing with the different
stages of parsing etc.

* pi: integrates all the parsing elements into the overall parser framework.

* langs: has the parsers for specific languages, including Go (of course), markdown
and tex (latter are lexer-only)

* filecat: is the core file categorization (e.g., mime-type, etc) framework that
associates files with their extensions and provides other type-level meta data
that enables the Pi system and GoGi and other GoKi framework tools to know how
to deal with different types of files.

Note that the GUI editor framework for creating and testing parsers is
in the Gide package: https://github.com/goki/gide  under the piv sub-package.
*/
package pi
