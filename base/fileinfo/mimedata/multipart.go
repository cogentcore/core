// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package mimedata

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/textproto"
)

// ToMultipart produces a MIME multipart representation of multiple data
// elements present in the stream -- this should be used in system.Clipboard
// whenever there are multiple elements to be pasted, because windows doesn't
// support multiple clip elements, and linux isn't very convenient either
func (mi Mimes) ToMultipart() []byte {
	var b bytes.Buffer
	mpw := multipart.NewWriter(io.Writer(&b))
	hdr := fmt.Sprintf("%v\n%v: multipart/mixed; boundary=%v\n", MIMEVersion1, ContentType, mpw.Boundary())
	b.Write(([]byte)(hdr))
	for _, d := range mi {
		mh := textproto.MIMEHeader{ContentType: {d.Type}}
		bin := false
		if !IsText(d.Type) {
			mh.Add(ContentTransferEncoding, "base64")
			bin = true
		}
		wp, _ := mpw.CreatePart(mh)
		if bin {
			eb := make([]byte, base64.StdEncoding.EncodedLen(len(d.Data)))
			base64.StdEncoding.Encode(eb, d.Data)
			wp.Write(eb)
		} else {
			wp.Write(d.Data)
		}
	}
	mpw.Close()
	return b.Bytes()
}

// FromMultipart parses a MIME multipart representation of multiple data
// elements into corresponding mime data components
func FromMultipart(body []byte, boundary string) Mimes {
	mi := make(Mimes, 0, 10)
	sr := bytes.NewReader(body)
	mr := multipart.NewReader(sr, boundary)
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			return mi
		}
		if err != nil {
			log.Printf("mimedata.IsMultipart: malformed multipart MIME: %v\n", err)
			return mi
		}
		b, err := io.ReadAll(p)
		if err != nil {
			log.Printf("mimedata.IsMultipart: bad ReadAll of multipart MIME: %v\n", err)
			return mi
		}
		d := Data{}
		d.Type = p.Header.Get(ContentType)
		cte := p.Header.Get(ContentTransferEncoding)
		if cte != "" {
			switch cte {
			case "base64":
				eb := make([]byte, base64.StdEncoding.DecodedLen(len(b)))
				n, _ := base64.StdEncoding.Decode(eb, b)
				b = eb[:n]
			}
		}
		d.Data = b
		mi = append(mi, &d)
	}
}
