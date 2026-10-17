// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"cogentcore.org/core/text/highlighting"
	_ "cogentcore.org/core/text/parse/supportedlanguages"
)

// The tests use the parse languages for syntax highlighting,
// which their expected results were made with.
func init() {
	highlighting.UseParse = true
}
