// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package websocket

import "github.com/gorilla/websocket"

// Client represents a WebSocket client connection.
// You can use [Connect] to create a new Client.
type Client struct {

	// conn is the underlying WebSocket connection.
	conn *websocket.Conn
}

// Connect connects to a WebSocket server and returns a [Client].
func Connect(url string) (*Client, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}
