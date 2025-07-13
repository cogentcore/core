// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package websocket

import (
	"cogentcore.org/core/base/errors"
	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client connection.
// You can use [Connect] to create a new Client.
type Client struct {

	// conn is the underlying WebSocket connection.
	conn *websocket.Conn

	// done is a channel that is closed when the connection is closed.
	done chan struct{}
}

// Connect connects to a WebSocket server and returns a [Client].
func Connect(url string) (*Client, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, done: make(chan struct{})}, nil
}

// OnMessage sets a callback function to be called when a message is received.
// This function can only be called once.
func (c *Client) OnMessage(f func(typ int, msg []byte)) {
	go func() {
		for {
			typ, msg, err := c.conn.ReadMessage()
			if errors.Log(err) != nil {
				close(c.done)
				return
			}
			f(typ, msg)
		}
	}()
}
