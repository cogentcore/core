// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagex

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/draw"
)

// jpegOrientation returns the EXIF orientation (1 to 8) of the given JPEG
// data, as set by cameras that save the pixels as the sensor has them,
// and say how to turn them to be upright, or 1 (as they are) if it has
// none or it cannot be read.
func jpegOrientation(b []byte) int {
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(b) {
		if b[i] != 0xFF {
			return 1
		}
		marker := b[i+1]
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0x01 {
			i += 2
			continue
		}
		if marker == 0xDA || marker == 0xD9 { // start of scan, end: no more metadata
			return 1
		}
		n := int(binary.BigEndian.Uint16(b[i+2:]))
		end := i + 2 + n
		if n < 2 || end > len(b) {
			return 1
		}
		seg := b[i+4 : end]
		if marker == 0xE1 && bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
			return exifOrientation(seg[6:])
		}
		i = end
	}
	return 1
}

// exifOrientation returns the orientation tag of the first IFD of the
// given TIFF data of EXIF, or 1 if it has none.
func exifOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	off := int(bo.Uint32(t[4:]))
	if off < 8 || off+2 > len(t) {
		return 1
	}
	count := int(bo.Uint16(t[off:]))
	for k := range count {
		e := off + 2 + 12*k
		if e+12 > len(t) {
			return 1
		}
		if bo.Uint16(t[e:]) == 0x0112 { // Orientation, a SHORT
			o := int(bo.Uint16(t[e+8:]))
			if o >= 1 && o <= 8 {
				return o
			}
			return 1
		}
	}
	return 1
}

// orient returns the given image turned upright for the given EXIF
// orientation: 2 mirrored, 3 turned 180°, 4 flipped, 5 transposed,
// 6 turned 90° clockwise, 7 transversed, and 8 turned 90° counterclockwise.
func orient(im image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return im
	}
	b := im.Bounds()
	w, h := b.Dx(), b.Dy()
	src := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(src, src.Bounds(), im, b.Min, draw.Src)
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := range h {
		for x := range w {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			si := src.PixOffset(x, y)
			di := dst.PixOffset(dx, dy)
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	return dst
}

// decodeOriented decodes the given image data in Go, turning JPEG images
// upright as their EXIF orientation says (see [jpegOrientation]), as
// browsers and image viewers show them.
func decodeOriented(b []byte) (image.Image, string, error) {
	im, format, err := image.Decode(bytes.NewReader(b))
	if err != nil || format != "jpeg" {
		return im, format, err
	}
	return orient(im, jpegOrientation(b)), format, nil
}
