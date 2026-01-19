// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package websocket

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/system"
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
	if system.GenerateHTMLArg() {
		return &Client{}, nil
	}
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, done: make(chan struct{})}, nil
}

// OnMessage sets a callback function to be called when a message is received.
// This function can only be called once.
func (c *Client) OnMessage(f func(typ MessageTypes, msg []byte)) {
	if system.GenerateHTMLArg() {
		return
	}
	go func() {
		for {
			typ, msg, err := c.conn.ReadMessage()
			if errors.Log(err) != nil {
				close(c.done)
				return
			}
			f(MessageTypes(typ), msg)
		}
	}()
}

// Send sends a message to the WebSocket server with the given type and message.
func (c *Client) Send(typ MessageTypes, msg []byte) error {
	if system.GenerateHTMLArg() {
		return nil
	}
	return c.conn.WriteMessage(int(typ), msg)
}

// Close cleanly closes the WebSocket connection.
// It does not directly trigger [Client.OnClose], but once the connection
// is closed, [Client.OnMessage] will trigger it.
func (c *Client) Close() error {
	if system.GenerateHTMLArg() {
		return nil
	}
	return c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

// OnClose sets a callback function to be called when the connection is closed.
// This function can only be called once.
func (c *Client) OnClose(f func()) {
	if system.GenerateHTMLArg() {
		return
	}
	go func() {
		<-c.done
		f()
	}()
}
