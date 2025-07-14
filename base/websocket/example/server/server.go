// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"

	"cogentcore.org/core/base/errors"
	"github.com/gorilla/websocket"
)

func main() {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return origin == "" || origin == "http://localhost:8080"
		},
	}

	conns := map[*websocket.Conn]struct{}{}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if errors.Log(err) != nil {
			return
		}
		conns[conn] = struct{}{}

		defer func() {
			conn.Close()
			delete(conns, conn)
		}()

		for {
			_, msg, err := conn.ReadMessage()
			if errors.Log(err) != nil {
				return
			}
			fmt.Println(string(msg))

			for other := range conns {
				if other == conn {
					continue
				}
				other.WriteMessage(websocket.TextMessage, msg)
			}
		}
	})

	http.ListenAndServe(":8081", nil)
}
