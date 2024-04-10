// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlview

import (
	_ "embed"
)

// UserAgentStyles contains the default user agent styles.
//
//go:embed html.css
var UserAgentStyles string
