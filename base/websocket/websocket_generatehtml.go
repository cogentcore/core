// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build generatehtml

package websocket

// Client is a no-op on generatehtml.
type Client struct{}

// Connect is a no-op on generatehtml.
func Connect(url string) (*Client, error) {
	return &Client{}, nil
}

// OnMessage is a no-op on generatehtml.
func (c *Client) OnMessage(f func(typ MessageTypes, msg []byte)) {}

// Send is a no-op on generatehtml.
func (c *Client) Send(typ MessageTypes, msg []byte) error {
	return nil
}

// Close is a no-op on generatehtml.
func (c *Client) Close() error {
	return nil
}

// OnClose is a no-op on generatehtml.
func (c *Client) OnClose(f func()) {}
