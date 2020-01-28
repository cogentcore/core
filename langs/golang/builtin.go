// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"strings"
	"unsafe"

	"github.com/goki/pi/complete"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
)

var BuiltinTypes syms.TypeMap

// InstallBuiltinTypes initializes the BuiltinTypes map
func InstallBuiltinTypes() {
	if len(BuiltinTypes) != 0 {
		return
	}
	for _, tk := range BuiltinTypeKind {
		ty := syms.NewType(tk.Name, tk.Kind)
		ty.Size = []int{tk.Size}
		BuiltinTypes.Add(ty)
	}
}

func (gl *GoLang) CompleteBuiltins(fs *pi.FileState, seed string, md *complete.Matches) {
	for _, tk := range BuiltinTypeKind {
		if strings.HasPrefix(tk.Name, seed) {
			c := complete.Completion{Text: tk.Name, Label: tk.Name, Icon: "type"}
			md.Matches = append(md.Matches, c)
		}
	}
	for _, bs := range BuiltinMisc {
		if strings.HasPrefix(bs, seed) {
			c := complete.Completion{Text: bs, Label: bs, Icon: "var"}
			md.Matches = append(md.Matches, c)
		}
	}
	for _, bs := range BuiltinFuncs {
		if strings.HasPrefix(bs, seed) {
			bs = bs + "()"
			c := complete.Completion{Text: bs, Label: bs, Icon: "function"}
			md.Matches = append(md.Matches, c)
		}
	}
	for _, bs := range BuiltinPackages {
		if strings.HasPrefix(bs, seed) {
			c := complete.Completion{Text: bs, Label: bs, Icon: "types"}
			md.Matches = append(md.Matches, c)
		}
	}
}

// BuiltinTypeKind are the type names and kinds for builtin Go primitive types
// (i.e., those with names)
var BuiltinTypeKind = []syms.TypeKindSize{
	{"int", syms.Int, int(unsafe.Sizeof(int(0)))},
	{"int8", syms.Int8, 1},
	{"int16", syms.Int16, 2},
	{"int32", syms.Int32, 4},
	{"int64", syms.Int64, 8},

	{"uint", syms.Uint, int(unsafe.Sizeof(uint(0)))},
	{"uint8", syms.Uint8, 1},
	{"uint16", syms.Uint16, 2},
	{"uint32", syms.Uint32, 4},
	{"uint64", syms.Uint64, 8},
	{"uintptr", syms.Uintptr, 8},

	{"byte", syms.Uint8, 1},
	{"rune", syms.Int32, 4},

	{"float32", syms.Float32, 4},
	{"float64", syms.Float64, 8},

	{"complex64", syms.Complex64, 8},
	{"complex128", syms.Complex128, 16},

	{"bool", syms.Bool, 1},

	{"string", syms.String, 0},

	{"error", syms.Interface, 0},

	{"struct{}", syms.Struct, 0},
	{"interface{}", syms.Interface, 0},
}

// BuiltinMisc are misc builtin items
var BuiltinMisc = []string{
	"true",
	"false",
}

// BuiltinFuncs are functions builtin to the Go language
var BuiltinFuncs = []string{
	"append",
	"copy",
	"delete",
	"len",
	"cap",
	"make",
	"new",
	"complex",
	"real",
	"imag",
	"close",
	"panic",
	"recover",
}

// BuiltinPackages are the standard library packages
var BuiltinPackages = []string{
	"bufio",
	"bytes",
	"context",
	"crypto",
	"compress",
	"encoding",
	"errors",
	"expvar",
	"flag",
	"fmt",
	"hash",
	"html",
	"image",
	"io",
	"log",
	"math",
	"mime",
	"net",
	"os",
	"path",
	"plugin",
	"reflect",
	"regexp",
	"runtime",
	"sort",
	"strconv",
	"strings",
	"sync",
	"syscall",
	"testing",
	"time",
	"unicode",
	"unsafe",
	"tar",
	"zip",
	"bzip2",
	"flate",
	"gzip",
	"lzw",
	"zlib",
	"heap",
	"list",
	"ring",
	"aes",
	"cipher",
	"des",
	"dsa",
	"ecdsa",
	"ed25519",
	"elliptic",
	"hmac",
	"md5",
	"rc4",
	"rsa",
	"sha1",
	"sha256",
	"sha512",
	"tls",
	"x509",
	"sql",
	"ascii85",
	"asn1",
	"base32",
	"base64",
	"binary",
	"csv",
	"gob",
	"hex",
	"json",
	"pem",
	"xml",
	"ast",
	"build",
	"constant",
	"doc",
	"format",
	"importer",
	"parser",
	"printer",
	"scanner",
	"token",
	"types",
	"adler32",
	"crc32",
	"crc64",
	"fnv",
	"template",
	"color",
	"draw",
	"gif",
	"jpeg",
	"png",
	"suffixarray",
	"ioutil",
	"syslog",
	"big",
	"bits",
	"cmplx",
	"rand",
	"multipart",
	"quotedprintable",
	"http",
	"cookiejar",
	"cgi",
	"httptrace",
	"httputil",
	"pprof",
	"socktest",
	"mail",
	"rpc",
	"jsonrpc",
	"smtp",
	"textproto",
	"url",
	"exec",
	"signal",
	"user",
	"filepath",
	"syntax",
	"cgo",
	"debug",
	"atomic",
	"math",
	"sys",
	"msan",
	"race",
	"trace",
	"atomic",
	"js",
	"scanner",
	"tabwriter",
	"template",
	"utf16",
	"utf8",
}
