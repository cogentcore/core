// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package main

import (
	_ "cogentcore.org/core/system/driver"
	"cogentcore.org/core/text/shaped/shapedjs"
)

func main() {
	shapedjs.MeasureTest()
}
