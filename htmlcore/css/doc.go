// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package css has the parts of CSS that htmlcore uses to style the
// widgets of pages that do not depend on its widgets: parsing style
// sheets ([Parse]), compiling and indexing selectors
// ([CompileSelector], [SelectorIndex]), the declarations of the
// cascade ([Origin], [ValidDeclaration]), and the values of
// properties: lengths ([LengthEnv]), colors ([ParseColor]), filters,
// transitions, fonts, masks and others. The cascade itself, which
// applies the declarations to the elements of a page as it is read,
// is in htmlcore.
package css
