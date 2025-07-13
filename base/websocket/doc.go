// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package websocket provides a cross-platform WebSocket client implementation.
// On native, it uses github.com/gorilla/websocket. On web, it uses the
// browser's built-in WebSocket support via syscall/js.
//
// The API is consistent across platforms, so low level details are not exposed.
package websocket
