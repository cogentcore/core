// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package coresymbols

// lazyLoad is whether [Load] is only called when the symbols are first
// needed instead of at init time (see [Load]).
const lazyLoad = false
