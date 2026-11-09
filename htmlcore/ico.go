// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
)

// The ICO format of favicons, the largest image of which is decoded: a
// PNG (as in most modern icons), or a BMP (a DIB of 32 bits per pixel with
// alpha, 24 bits, or a palette of 1, 4 or 8 bits, with a mask of the
// transparent pixels), as the favicons of github.com that duckduckgo.com
// shows by its results.

func init() {
	image.RegisterFormat("ico", "\x00\x00\x01\x00", decodeICO, decodeICOConfig)
}

// icoImage returns the data of the largest image in the given ICO file,
// its size, and whether it is a PNG (or else a DIB).
func icoImage(r io.Reader) (img []byte, w, h int, isPNG bool, err error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, 0, false, err
	}
	if len(data) < 6 {
		return nil, 0, 0, false, errors.New("ico: too short")
	}
	n := int(binary.LittleEndian.Uint16(data[4:6]))
	bestArea := -1
	for i := range n {
		e := 6 + 16*i
		if e+16 > len(data) {
			break
		}
		ew, eh := int(data[e]), int(data[e+1])
		if ew == 0 {
			ew = 256
		}
		if eh == 0 {
			eh = 256
		}
		size := int(binary.LittleEndian.Uint32(data[e+8 : e+12]))
		off := int(binary.LittleEndian.Uint32(data[e+12 : e+16]))
		if off < 0 || size <= 0 || off+size > len(data) || ew*eh <= bestArea {
			continue
		}
		img, w, h, isPNG, bestArea = data[off:off+size], ew, eh, bytes.HasPrefix(data[off:], []byte("\x89PNG")), ew*eh
	}
	if img == nil {
		return nil, 0, 0, false, errors.New("ico: no image")
	}
	return img, w, h, isPNG, nil
}

func decodeICO(r io.Reader) (image.Image, error) {
	b, w, h, isPNG, err := icoImage(r)
	if err != nil {
		return nil, err
	}
	if isPNG {
		return png.Decode(bytes.NewReader(b))
	}
	return decodeDIB(b, w, h)
}

func decodeICOConfig(r io.Reader) (image.Config, error) {
	b, w, h, isPNG, err := icoImage(r)
	if err != nil {
		return image.Config{}, err
	}
	if isPNG {
		return png.DecodeConfig(bytes.NewReader(b))
	}
	return image.Config{ColorModel: color.NRGBAModel, Width: w, Height: h}, nil
}

// decodeDIB decodes the given DIB of an ICO file (a BMP with no file
// header, whose height is that of its image and of its mask), of the given
// size: its rows are from the bottom, of 32, 24, 8, 4 or 1 bits per pixel
// (with a palette for 8 and fewer), padded to 4 bytes, followed by the
// mask of 1 bit per pixel (1 for a transparent pixel).
func decodeDIB(b []byte, w, h int) (image.Image, error) {
	if len(b) < 40 {
		return nil, errors.New("ico: short BMP header")
	}
	hdr := int(binary.LittleEndian.Uint32(b[0:4]))
	if dw := int(int32(binary.LittleEndian.Uint32(b[4:8]))); dw > 0 {
		w = dw
	}
	if dh := int(int32(binary.LittleEndian.Uint32(b[8:12]))); dh > 0 {
		h = dh / 2 // with the mask
	}
	bpp := int(binary.LittleEndian.Uint16(b[14:16]))
	if c := binary.LittleEndian.Uint32(b[16:20]); c != 0 && c != 3 { // BI_RGB (or bitfields, as BGRA)
		return nil, errors.New("ico: compressed BMP")
	}
	var palette []color.NRGBA
	pos := hdr
	if bpp <= 8 {
		n := int(binary.LittleEndian.Uint32(b[32:36]))
		if n == 0 {
			n = 1 << bpp
		}
		for i := 0; i < n && pos+4 <= len(b); i++ {
			palette = append(palette, color.NRGBA{b[pos+2], b[pos+1], b[pos], 255})
			pos += 4
		}
	}
	stride := (w*bpp + 31) / 32 * 4
	maskStride := (w + 31) / 32 * 4
	if pos+stride*h > len(b) {
		return nil, errors.New("ico: short BMP")
	}
	mask := b[pos+stride*h:]
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	hasAlpha := false
	for y := range h {
		row := b[pos+stride*(h-1-y):]
		for x := range w {
			var c color.NRGBA
			switch bpp {
			case 32:
				c = color.NRGBA{row[4*x+2], row[4*x+1], row[4*x], row[4*x+3]}
				hasAlpha = hasAlpha || c.A != 0
			case 24:
				c = color.NRGBA{row[3*x+2], row[3*x+1], row[3*x], 255}
			case 8, 4, 1:
				bit := x * bpp
				i := int(row[bit/8]>>(8-bpp-bit%8)) & (1<<bpp - 1)
				if i < len(palette) {
					c = palette[i]
				}
			default:
				return nil, errors.New("ico: unsupported BMP bits per pixel")
			}
			img.SetNRGBA(x, y, c)
		}
	}
	if bpp == 32 && hasAlpha {
		return img, nil // its alpha, which the mask is not needed for
	}
	for y := range h {
		mrow := maskStride * (h - 1 - y)
		for x := range w {
			i := mrow + x/8
			transparent := i < len(mask) && mask[i]&(0x80>>(x%8)) != 0
			c := img.NRGBAAt(x, y)
			c.A = 255
			if transparent {
				c.A = 0
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img, nil
}
