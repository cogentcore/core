// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mimedata defines MIME data support used in clipboard and
// drag-and-drop functions in the GoGi GUI.  mimedata.Data struct contains
// format and []byte data, and multiple representations of the same data are
// encoded in mimedata.Mimes which is just a []mimedata.Data slice -- it can
// be encoded / decoded from mime multipart.
//
// see filecat package for known mimetypes
//
package mimedata

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"

	"github.com/goki/pi/filecat"
)

const (
	MIMEVersion1            = "MIME-Version: 1.0"
	ContentType             = "Content-Type"
	ContentTransferEncoding = "Content-Transfer-Encoding"
)

var MIMEVersion1B = ([]byte)(MIMEVersion1)
var ContentTypeB = ([]byte)(ContentType)
var ContentTransferEncodingB = ([]byte)(ContentTransferEncoding)

// Data represents one element of MIME data as a type string and byte slice
type Data struct {
	// MIME Type string representing the data, e.g., text/plain, text/html, text/xml, text/uri-list, image/jpg, png etc
	Type string

	// Data for the item
	Data []byte
}

// NewTextData returns a Data representation of the string -- good idea to
// always have a text/plain representation of everything on clipboard /
// drag-n-drop
func NewTextData(text string) *Data {
	return &Data{filecat.TextPlain, []byte(text)}
}

// NewTextDataBytes returns a Data representation of the bytes string
func NewTextDataBytes(text []byte) *Data {
	return &Data{filecat.TextPlain, text}
}

// IsText returns true if type is any of the text/ types (literally looks for that
// at start of Type) or is another known text type (e.g., AppJSON, XML)
func IsText(typ string) bool {
	if strings.HasPrefix(typ, "text/") {
		return true
	}
	if typ == filecat.DataJson || typ == filecat.DataXml {
		return true
	}
	return false
}

////////////////////////////////////////////////////////////////////////////////
//    Mimes

// Mimes is a slice of mime data, potentially encoding the same data in
// different formats -- this is used for all oswin API's for maximum
// flexibility
type Mimes []*Data

// NewMimes returns a new Mimes slice of given length and capacity
func NewMimes(ln, cp int) Mimes {
	return make(Mimes, ln, cp)
}

// NewText returns a Mimes representation of the string as a single text/plain Data
func NewText(text string) Mimes {
	md := NewTextData(text)
	mi := make(Mimes, 1)
	mi[0] = md
	return mi
}

// NewTextBytes returns a Mimes representation of the bytes string as a single text/plain Data
func NewTextBytes(text []byte) Mimes {
	md := NewTextDataBytes(text)
	mi := make(Mimes, 1)
	mi[0] = md
	return mi
}

// NewTextPlus returns a Mimes representation of an item as a text string plus
// a more specific type
func NewTextPlus(text, typ string, data []byte) Mimes {
	md := NewTextData(text)
	mi := make(Mimes, 2)
	mi[0] = md
	mi[1] = &Data{typ, data}
	return mi
}

// NewMime returns a Mimes representation of one element
func NewMime(typ string, data []byte) Mimes {
	mi := make(Mimes, 1)
	mi[0] = &Data{typ, data}
	return mi
}

// HasType returns true if Mimes has given type of data available
func (mi Mimes) HasType(typ string) bool {
	for _, d := range mi {
		if d.Type == typ {
			return true
		}
	}
	return false
}

// TypeData returns data associated with given MIME type
func (mi Mimes) TypeData(typ string) []byte {
	for _, d := range mi {
		if d.Type == typ {
			return d.Data
		}
	}
	return nil
}

// Text extracts all the text elements of given type as a string
func (mi Mimes) Text(typ string) string {
	str := ""
	for _, d := range mi {
		if d.Type == typ {
			str = str + string(d.Data)
		}
	}
	return str
}

// ToMultipart produces a MIME multipart representation of multiple data
// elements present in the stream -- this should be used in clip.Board
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

// IsMultipart examines data bytes to see if it has a MIME-Version: 1.0
// ContentType: multipart/* header -- returns the actual multipart media type,
// body of the data string after the header (assumed to be a single \n
// terminated line at start of string, and the boundary separating multipart
// elements (all from mime.ParseMediaType) -- mediaType is the mediaType if it
// is another MIME type -- can check that for non-empty string
func IsMultipart(str []byte) (isMulti bool, mediaType, boundary string, body []byte) {
	isMulti = false
	mediaType = ""
	boundary = ""
	body = ([]byte)("")
	var pars map[string]string
	var err error
	if bytes.HasPrefix(str, MIMEVersion1B) {
		cri := bytes.IndexByte(str, '\n')
		if cri < 0 { // shouldn't happen
			return
		}
		ctln := str[cri+1:]
		if bytes.HasPrefix(ctln, ContentTypeB) { // should
			cri2 := bytes.IndexByte(ctln, '\n')
			if cri2 < 0 { // shouldn't happen
				return
			}
			hdr := ctln[len(ContentTypeB)+1 : cri2]
			mediaType, pars, err = mime.ParseMediaType(string(hdr))
			if err != nil { // shouldn't happen
				log.Printf("mimedata.IsMultipart: malformed MIME header: %v\n", err)
				return
			}
			if strings.HasPrefix(mediaType, "multipart/") {
				isMulti = true
				body = str[cri2+1:]
				boundary = pars["boundary"]
			}
		}
	}
	return
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
		b, err := ioutil.ReadAll(p)
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
				base64.StdEncoding.Decode(eb, b)
				b = eb
			}
		}
		d.Data = b
		mi = append(mi, &d)
	}
	return mi
}

// todo: image, etc extractors
