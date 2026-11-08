// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// icoBMP returns an ICO file of one 2x2 BMP image of the given bits per
// pixel, palette, pixel rows (from the bottom, padded) and mask rows.
func icoBMP(bpp int, palette, pixels, mask []byte) []byte {
	var dib bytes.Buffer
	hdr := make([]byte, 40)
	binary.LittleEndian.PutUint32(hdr[0:], 40)
	binary.LittleEndian.PutUint32(hdr[4:], 2)
	binary.LittleEndian.PutUint32(hdr[8:], 4) // with the mask
	binary.LittleEndian.PutUint16(hdr[12:], 1)
	binary.LittleEndian.PutUint16(hdr[14:], uint16(bpp))
	binary.LittleEndian.PutUint32(hdr[32:], uint32(len(palette)/4))
	dib.Write(hdr)
	dib.Write(palette)
	dib.Write(pixels)
	dib.Write(mask)
	var ico bytes.Buffer
	ico.Write([]byte{0, 0, 1, 0, 1, 0})
	entry := make([]byte, 16)
	entry[0], entry[1] = 2, 2
	binary.LittleEndian.PutUint32(entry[8:], uint32(dib.Len()))
	binary.LittleEndian.PutUint32(entry[12:], 22)
	ico.Write(entry)
	ico.Write(dib.Bytes())
	return ico.Bytes()
}

// TestICOBMP tests icons of BMP images (as the favicons of github.com that
// duckduckgo.com shows by its results, which were missing): of 32 bits per
// pixel with alpha, and of 8 bits with a palette and a mask.
func TestICOBMP(t *testing.T) {
	// 32 bits: the bottom row is red and transparent, the top green and blue
	pixels := []byte{0, 0, 255, 255, 0, 0, 0, 0, 0, 255, 0, 255, 255, 0, 0, 255}
	mask := make([]byte, 8)
	img, kind, err := image.Decode(bytes.NewReader(icoBMP(32, nil, pixels, mask)))
	require.NoError(t, err)
	assert.Equal(t, "ico", kind)
	at := func(x, y int) color.NRGBA { return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA) }
	assert.Equal(t, color.NRGBA{0, 255, 0, 255}, at(0, 0))
	assert.Equal(t, color.NRGBA{0, 0, 255, 255}, at(1, 0))
	assert.Equal(t, color.NRGBA{255, 0, 0, 255}, at(0, 1))
	assert.Equal(t, uint8(0), at(1, 1).A, "transparent")

	// 8 bits: a palette of black and white, with the right column masked
	palette := []byte{0, 0, 0, 0, 255, 255, 255, 0}
	pixels = []byte{1, 0, 0, 0, 0, 1, 0, 0} // bottom: white, black; top: black, white
	mask = []byte{0x40, 0, 0, 0, 0x40, 0, 0, 0}
	img, _, err = image.Decode(bytes.NewReader(icoBMP(8, palette, pixels, mask)))
	require.NoError(t, err)
	assert.Equal(t, color.NRGBA{0, 0, 0, 255}, at(0, 0))
	assert.Equal(t, color.NRGBA{255, 255, 255, 255}, at(0, 1))
	assert.Equal(t, uint8(0), at(1, 0).A, "masked")
	assert.Equal(t, uint8(0), at(1, 1).A, "masked")
}

// TestFaviconImages tests the favicons of the results of duckduckgo.com:
// an ICO of a BMP image (which [imagex.Read] decodes with an error, as ICO
// is not one of its formats), and the default icon, a PNG with a status of
// 404, which browsers show.
func TestFaviconImages(t *testing.T) {
	pixels := []byte{0, 0, 255, 255, 0, 0, 255, 255, 0, 0, 255, 255, 0, 0, 255, 255}
	ico := icoBMP(32, nil, pixels, make([]byte, 8))
	var pngData bytes.Buffer
	require.NoError(t, png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	ctx := NewContext()
	ctx.GetURL = func(u string) (*http.Response, error) {
		switch {
		case strings.HasSuffix(u, "github.com.ico"):
			return &http.Response{StatusCode: 200, Status: "200 OK", Header: http.Header{"Content-Type": {"image/x-icon"}}, Body: io.NopCloser(bytes.NewReader(ico))}, nil
		default:
			return &http.Response{StatusCode: 404, Status: "404 Not Found", Header: http.Header{"Content-Type": {"image/png"}}, Body: io.NopCloser(bytes.NewReader(pngData.Bytes()))}, nil
		}
	}
	b := core.NewBody()
	require.NoError(t, ReadHTMLString(ctx, b, `<img src="https://icons.test/github.com.ico" width="16" height="16"><img src="https://icons.test/cogentcore.org.ico" width="16" height="16">`))
	var imgs []*core.Image
	b.WalkDown(func(k tree.Node) bool {
		if im, ok := k.(*core.Image); ok {
			imgs = append(imgs, im)
		}
		return tree.Continue
	})
	require.Len(t, imgs, 2)
	assert.NotNil(t, imgs[0].Image, "the ICO of a BMP")
	assert.NotNil(t, imgs[1].Image, "the PNG of the 404")
}
