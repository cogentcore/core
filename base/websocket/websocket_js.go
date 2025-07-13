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

// Connect connects to a WebSocket server and returns a [Client].
func Connect(url string) (*Client, error) {
	ws := js.Global().Get("WebSocket").New(url)
	ws.Set("binaryType", "arraybuffer")
	return &Client{ws: ws}, nil
}

// OnMessage sets a callback function to be called when a message is received.
// This function can only be called once on native.
func (c *Client) OnMessage(f func(typ MessageTypes, msg []byte)) {
	c.ws.Call("addEventListener", "message", js.FuncOf(func(this js.Value, args []js.Value) any {
		data := args[0].Get("data")
		if data.Type() == js.TypeString {
			f(TextMessage, []byte(data.String()))
			return nil
		}
		array := js.Global().Get("Uint8ClampedArray").New(data)
		b := make([]byte, array.Length())
		js.CopyBytesToGo(b, array) // TODO: more performant way to do this, perhaps with gopherjs/goscript?
		f(BinaryMessage, b)
		return nil
	}))
}
