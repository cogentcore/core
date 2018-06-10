// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x11driver

import (
	"log"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/goki/gi/oswin/mimedata"
)

// implements clipboard support for X11
// https://github.com/jtanx/libclipboard/blob/master/src/clipboard_x11.c
// https://www.uninformativ.de/blog/postings/2017-04-02/0/POSTING-en.html

type clipImpl struct {
	data mimedata.Mimes
	app  *appImpl
}

var theClip = clipImpl{}

func (ci *clipImpl) Read(types []string) mimedata.Mimes {
	if types == nil {
		return nil
	}
	ci.data = nil

	// check for an owner on either the PRIMARY or CLIPBOARD selections
	// owner info is not actually used under the XCB/XGB protocol -- just to see
	// that it exists..
	useSel := ci.app.atomPrimarySel
	selown, err := xproto.GetSelectionOwner(ci.app.xc, ci.app.atomPrimarySel).Reply()
	if err != nil {
		log.Printf("X11 Clipboard Read error: %v\n", err)
		return nil
	}
	if selown.Owner == xproto.AtomNone {
		useSel = ci.app.atomClipboardSel
		selown, err = xproto.GetSelectionOwner(ci.app.xc, ci.app.atomClipboardSel).Reply()
		if err != nil {
			log.Printf("X11 Clipboard Read error: %v\n", err)
			return nil
		}
	}
	if selown.Owner == xproto.AtomNone { // nothing there..
		return nil
	}

	// this is the main call requesting the selection -- there are no apparent
	// docs for the xcb version of this call, in terms of the "Property" arg,
	// but example from jtanx just uses the name of the selection again, so...
	xproto.ConvertSelection(ci.app.xc, ci.app.window32, useSel, ci.app.atomUTF8String, useSel, xproto.TimeCurrentTime)

	transSz := uint32(1048576) // how much to transfer per call, 1MB
	var ptyp xproto.Atom
	b := make([]byte, 0, 1024)

	// todo: need some kind of timeout on this!
	ev := <-ci.app.selNotifyChan
	bytesAfter := uint32(1)
	bufsz := uint32(0) // current buffer size
	for bytesAfter > 0 {
		// last two args are offset and amount to transfer, in 32bit "long" sizes
		prop, err := xproto.GetProperty(ci.app.xc, true, ci.app.window32, ev.Property, xproto.AtomAny, bufsz/4, transSz/4).Reply()
		if err != nil {
			log.Printf("X11 Clipboard Read Property error: %v\n", err)
			return nil
		}
		bytesAfter = prop.BytesAfter
		sz := len(prop.Value)
		if sz > 0 {
			b = append(b, prop.Value...)
			bufsz += uint32(sz)
		}
		ptyp = prop.Type
	}

	// fmt.Printf("ptyp: %v and utf8: %v \n", ptyp, ci.app.atomUTF8String)
	for _, typ := range types {
		if typ == mimedata.TextPlain && ptyp == ci.app.atomUTF8String {
			ci.data = mimedata.NewText(string(b))
		}
	}
	return ci.data
}

// func clipWriteMimeType(mtyp string) {
// 	ctyp := C.CString(mimedata.TextPlain)
// 	mhdr := []byte(fmt.Sprintf("MIME-Version: 1.0\nContent-type: %v", mtyp))
// 	sz := len(mhdr)
// 	cdata := C.malloc(C.size_t(sz))
// 	copy((*[1 << 30]byte)(cdata)[0:sz], mhdr)
// 	C.clipAddWrite((*C.char)(cdata), C.int(sz), ctyp, C.int(len(mimedata.TextPlain)))
// 	C.free(unsafe.Pointer(ctyp))
// 	C.free(unsafe.Pointer(cdata))
// }

func (ci *clipImpl) Write(data mimedata.Mimes, clearFirst bool) error {
	if clearFirst {
		ci.Clear()
	}
	return nil
}

// 	for _, d := range data {
// 		switch d.Type {
// 		case mimedata.AppJSON:
// 			clipWriteMimeType(d.Type)
// 		}
// 		ctyp := C.CString(d.Type)
// 		sz := len(d.Data)
// 		cdata := C.malloc(C.size_t(sz))
// 		copy((*[1 << 30]byte)(cdata)[0:sz], d.Data)
// 		C.clipAddWrite((*C.char)(cdata), C.int(sz), ctyp, C.int(len(d.Type)))
// 		C.free(unsafe.Pointer(ctyp))
// 		C.free(unsafe.Pointer(cdata))
// 	}
// 	C.clipWrite()
// 	return nil
// }

func (ci *clipImpl) Clear() {
	// 	C.clipClear()
}

// //export addMimeData
// func addMimeData(ctyp *C.char, typlen C.int, cdata *C.char, datalen C.int) {
// 	if *curMimeData == nil {
// 		*curMimeData = make(mimedata.Mimes, 0)
// 	}
// 	typ := C.GoStringN(ctyp, typlen)
// 	data := C.GoBytes(unsafe.Pointer(cdata), datalen)
// 	*curMimeData = append(*curMimeData, &mimedata.Data{typ, data})
// }
