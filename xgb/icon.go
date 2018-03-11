/*
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

package xgb

import (
	"bytes"
	"github.com/BurntSushi/xgbutil/ewmh"
	"image"
	"image/gif"
)

var Gordon image.Image

func init() {
	gordonGifData := gordon_gif()
	var err error
	Gordon, err = gif.Decode(bytes.NewReader(gordonGifData))
	if err != nil {
		panic(err)
	}
}

func (w *OSWindow) SetIconName(name string) {
	err := ewmh.WmIconNameSet(w.xu, w.win.Id, name)
	if err != nil {
		println(err.Error())
	}
}

func (w *OSWindow) SetIcon(icon image.Image) {
	width := icon.Bounds().Max.X - icon.Bounds().Min.X
	height := icon.Bounds().Max.Y - icon.Bounds().Min.Y
	data := make([]uint, width*height)
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			i := x + y*width
			c := icon.At(x, y)
			r, g, b, a := c.RGBA()
			data[i] = uint(a + r<<8 + g<<16 + b<<24)
		}
	}
	wmicon := ewmh.WmIcon{
		Width:  uint(width),
		Height: uint(height),
		Data:   data,
	}
	ewmh.WmIconSet(w.xu, w.win.Id, []ewmh.WmIcon{wmicon})
}
