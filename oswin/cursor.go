// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oswin

/*
   directly copied from https://github.com/skelterjohn/go.wde

   Copyright 2012 the go.wde authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

type Cursor int

const (
	NoneCursor Cursor = iota
	NormalCursor
	ResizeNCursor
	ResizeECursor
	ResizeSCursor
	ResizeWCursor
	ResizeEWCursor
	ResizeNSCursor
	ResizeNECursor
	ResizeSECursor
	ResizeSWCursor
	ResizeNWCursor
	CrosshairCursor
	IBeamCursor
	GrabHoverCursor
	GrabActiveCursor
	NotAllowedCursor
	customCursorBase // custom cursors are numbered starting here
)

type CursorCtl interface {
	Set(id Cursor)
	Hide()
	Show()
}

/* TODO: custom cursors: func CreateCursor(draw.Image, hotspot image.Point) Cursor

func (c Cursor) IsCustom() bool {
	return c >= customCursorBase
}
*/
