// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/png"
	"io"
)

// The ICO format of favicons, for its images that are stored as PNG
// (as in most modern icons), the largest of which is decoded.

func init() {
	image.RegisterFormat("ico", "\x00\x00\x01\x00", decodeICO, decodeICOConfig)
}

// icoPNG returns the data of the largest PNG image in the given ICO file.
func icoPNG(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(data) < 6 {
		return nil, errors.New("ico: too short")
	}
	n := int(binary.LittleEndian.Uint16(data[4:6]))
	var best []byte
	bestArea := -1
	for i := range n {
		e := 6 + 16*i
		if e+16 > len(data) {
			break
		}
		w, h := int(data[e]), int(data[e+1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		size := int(binary.LittleEndian.Uint32(data[e+8 : e+12]))
		off := int(binary.LittleEndian.Uint32(data[e+12 : e+16]))
		if off < 0 || size <= 0 || off+size > len(data) {
			continue
		}
		img := data[off : off+size]
		if bytes.HasPrefix(img, []byte("\x89PNG")) && w*h > bestArea {
			best, bestArea = img, w*h
		}
	}
	if best == nil {
		return nil, errors.New("ico: no PNG image (BMP images are not supported)")
	}
	return best, nil
}

func decodeICO(r io.Reader) (image.Image, error) {
	b, err := icoPNG(r)
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(b))
}

func decodeICOConfig(r io.Reader) (image.Config, error) {
	b, err := icoPNG(r)
	if err != nil {
		return image.Config{}, err
	}
	return png.DecodeConfig(bytes.NewReader(b))
}
