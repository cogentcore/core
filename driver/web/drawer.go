// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

// drawerImpl is a TEMPORARY, low-performance implementation of [goosi.Drawer].
// It will be replaced with a full WebGPU based drawer at some point.
// TODO: replace drawerImpl with WebGPU
type drawerImpl struct{}
