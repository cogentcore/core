// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/base/websocket"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
)

func main() {
	b := core.NewBody("Websocket Example")

	ws, err := websocket.Connect("ws://localhost:8081/ws")
	core.ErrorDialog(b, err)

	msgs := []string{}
	var list *core.List

	tf := core.NewTextField(b).SetPlaceholder("Enter message")
	send := core.NewButton(b).SetText("Send")
	send.OnClick(func(e events.Event) {
		msg := tf.Text()
		ws.Send(websocket.TextMessage, []byte(msg))
		msgs = append(msgs, msg)
		list.Update()
	})

	list = core.NewList(b)
	list.SetSlice(&msgs).SetReadOnly(true)

	ws.OnMessage(func(typ websocket.MessageTypes, msg []byte) {
		list.AsyncLock()
		defer list.AsyncUnlock()
		msgs = append(msgs, string(msg))
		list.Update()
	})

	b.RunMainWindow()
}
