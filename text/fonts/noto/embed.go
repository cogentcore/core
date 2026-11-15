// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package noto embeds the Noto Sans fonts, the default sans-serif fonts.
package noto

import "embed"

//go:embed *.ttf
var Embedded embed.FS
