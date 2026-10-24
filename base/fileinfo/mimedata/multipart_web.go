// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mimedata

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// The web versions of [Mimes.ToMultipart] and [FromMultipart] read and write
// the MIME multipart format without the mime/multipart package, which needs
// net/textproto and with it the net package (most of a megabyte of a web app).
// They write the same format as mime/multipart, and read what it writes.

// writeMultipart returns the MIME multipart representation of the given data
// with a new random boundary, as [Mimes.ToMultipart] does with mime/multipart.
func writeMultipart(mi Mimes) []byte {
	// A random boundary, as made by multipart.Writer.
	var rb [30]byte
	rand.Read(rb[:])
	boundary := hex.EncodeToString(rb[:])

	// The header line, then each part with its sorted header fields.
	var b bytes.Buffer
	fmt.Fprintf(&b, "%v\n%v: multipart/mixed; boundary=%v\n", MIMEVersion1, ContentType, boundary)
	for i, d := range mi {
		if i > 0 {
			b.WriteString("\r\n")
		}
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		data := d.Data
		if !IsText(d.Type) {
			fmt.Fprintf(&b, "%s: base64\r\n", ContentTransferEncoding)
			data = []byte(base64.StdEncoding.EncodeToString(d.Data))
		}
		fmt.Fprintf(&b, "%s: %s\r\n\r\n", ContentType, d.Type)
		b.Write(data)
	}

	// The closing boundary.
	fmt.Fprintf(&b, "\r\n--%s--\r\n", boundary)
	return b.Bytes()
}

// readMultipart parses the MIME multipart body with the given boundary, as
// [FromMultipart] does with mime/multipart.
func readMultipart(body []byte, boundary string) Mimes {
	mi := make(Mimes, 0, 10)
	str := string(body)
	delim := "--" + boundary

	// Skip the preamble up to the first boundary line.
	start := strings.Index(str, delim)
	if start < 0 {
		return mi
	}
	str = str[start:]
	for {
		// Each boundary line ends the previous part; "--" ends the last one.
		str = str[len(delim):]
		if strings.HasPrefix(str, "--") {
			return mi
		}
		nl := strings.IndexByte(str, '\n')
		if nl < 0 {
			return mi
		}
		str = str[nl+1:]

		// The content of the part ends at the line break before the next boundary.
		end := strings.Index(str, "\n"+delim)
		if end < 0 {
			return mi
		}
		part := strings.TrimSuffix(str[:end], "\r")
		str = str[end+1:]
		mi = append(mi, readPart(part))
	}
}

// readPart returns the data of one part of a MIME multipart body:
// its header fields, an empty line, then its content.
func readPart(part string) *Data {
	d := &Data{}
	cte := ""
	for {
		// Each header line until the empty line.
		nl := strings.IndexByte(part, '\n')
		if nl < 0 {
			nl = len(part) - 1
		}
		line := strings.TrimRight(part[:nl+1], "\r\n")
		part = part[nl+1:]
		if line == "" {
			break
		}
		key, value, _ := strings.Cut(line, ":")
		value = strings.TrimSpace(value)
		switch {
		case strings.EqualFold(key, ContentType):
			d.Type = value
		case strings.EqualFold(key, ContentTransferEncoding):
			cte = value
		}
	}

	// The content, base64 decoded as by FromMultipart.
	b := []byte(part)
	if cte == "base64" {
		eb := make([]byte, base64.StdEncoding.DecodedLen(len(b)))
		base64.StdEncoding.Decode(eb, b)
		b = eb
	}
	d.Data = b
	return d
}
