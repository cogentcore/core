// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

// GenerateHTML is the function called for the -generatehtml argument,
// which the core tool uses to pre-render apps as HTML for a preview
// and for SEO purposes. It returns the generated HTML for the given widget.
// It exits the program if there is an error, but does not exit
// the program if there is no error. It is set by importing
// cogentcore.org/core/htmlcore/tohtml (which the core tool links into
// the app when it runs it with -generatehtml), and it may be replaced
// by packages with custom output (such as content).
var GenerateHTML func(w Widget) string
