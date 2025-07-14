// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package websocket provides a cross-platform WebSocket client implementation.
// On native, it uses github.com/gorilla/websocket. On web, it uses the
// browser's built-in WebSocket support via syscall/js.
//
// The API is consistent across platforms, so low level details are not exposed.
// Use [Connect] to make a [Client].
package websocket

//go:generate core generate

// MessageTypes are the types of messages that can be sent or received.
type MessageTypes int32 //enums:enum

const (

	// TextMessage is a text message (string).
	// Equivalent to [websocket.TextMessage].
	TextMessage MessageTypes = iota + 1

	// BinaryMessage is a binary data message ([]byte).
	// Equivalent to [websocket.BinaryMessage].
	BinaryMessage
)
