// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package supportedlanguages includes all the supported languages for parse -- need to
// import this package to get those all included in a given target
package supportedlanguages

import (
	_ "cogentcore.org/core/text/parse/languages/golang"
	_ "cogentcore.org/core/text/parse/languages/markdown"
	_ "cogentcore.org/core/text/parse/languages/tex"
)
