// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package websocket

import "syscall/js"

// Client represents a WebSocket client connection.
// You can use [Connect] to create a new Client.
type Client struct {

	// ws is the underlying JavaScript WebSocket object.
	// See https://developer.mozilla.org/en-US/docs/Web/API/WebSocket
	ws js.Value
}
