// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"strings"
	"sync"
	"unsafe"

	"cogentcore.org/core/icons"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/parse/syms"
)

var BuiltinTypes syms.TypeMap

// installBuiltinTypesOnce makes InstallBuiltinTypes initialize
// the BuiltinTypes map only once, as parsing goroutines call it at once.
var installBuiltinTypesOnce sync.Once

// InstallBuiltinTypes initializes the BuiltinTypes map
func InstallBuiltinTypes() {
	installBuiltinTypesOnce.Do(func() {
		for _, tk := range BuiltinTypeKind {
			ty := syms.NewType(tk.Name, tk.Kind)
			ty.Size = []int{tk.Size}
			BuiltinTypes.Add(ty)
		}
	})
}

func (gl *GoLang) CompleteBuiltins(fs *parse.FileState, seed string, md *complete.Matches) {
	for _, tk := range BuiltinTypeKind {
		if strings.HasPrefix(tk.Name, seed) {
			c := complete.Completion{Text: tk.Name, Label: tk.Name, Icon: icons.Type}
			md.Matches = append(md.Matches, c)
		}
	}
	for _, bs := range BuiltinMisc {
		if strings.HasPrefix(bs, seed) {
			c := complete.Completion{Text: bs, Label: bs, Icon: icons.Variable}
			md.Matches = append(md.Matches, c)
		}
	}
	for _, bs := range BuiltinFuncs {
		if strings.HasPrefix(bs, seed) {
			bs = bs + "()"
			c := complete.Completion{Text: bs, Label: bs, Icon: icons.Function}
			md.Matches = append(md.Matches, c)
		}
	}
	for _, bs := range BuiltinPackages {
		if strings.HasPrefix(bs, seed) {
			c := complete.Completion{Text: bs, Label: bs, Icon: icons.Type} // todo: was types
			md.Matches = append(md.Matches, c)
		}
	}
	for _, bs := range BuiltinKeywords {
		if strings.HasPrefix(bs, seed) {
			c := complete.Completion{Text: bs, Label: bs, Icon: icons.Constant}
			md.Matches = append(md.Matches, c)
		}
	}
}

// BuiltinTypeKind are the type names and kinds for builtin Go primitive types
// (i.e., those with names)
var BuiltinTypeKind = []syms.TypeKindSize{
	{Name: "int", Kind: syms.Int, Size: int(unsafe.Sizeof(int(0)))},
	{Name: "int8", Kind: syms.Int8, Size: 1},
	{Name: "int16", Kind: syms.Int16, Size: 2},
	{Name: "int32", Kind: syms.Int32, Size: 4},
	{Name: "int64", Kind: syms.Int64, Size: 8},

	{Name: "uint", Kind: syms.Uint, Size: int(unsafe.Sizeof(uint(0)))},
	{Name: "uint8", Kind: syms.Uint8, Size: 1},
	{Name: "uint16", Kind: syms.Uint16, Size: 2},
	{Name: "uint32", Kind: syms.Uint32, Size: 4},
	{Name: "uint64", Kind: syms.Uint64, Size: 8},
	{Name: "uintptr", Kind: syms.Uintptr, Size: 8},

	{Name: "byte", Kind: syms.Uint8, Size: 1},
	{Name: "rune", Kind: syms.Int32, Size: 4},

	{Name: "float32", Kind: syms.Float32, Size: 4},
	{Name: "float64", Kind: syms.Float64, Size: 8},

	{Name: "complex64", Kind: syms.Complex64, Size: 8},
	{Name: "complex128", Kind: syms.Complex128, Size: 16},

	{Name: "bool", Kind: syms.Bool, Size: 1},

	{Name: "string", Kind: syms.String, Size: 0},

	{Name: "error", Kind: syms.Interface, Size: 0},

	{Name: "struct{}", Kind: syms.Struct, Size: 0},
	{Name: "interface{}", Kind: syms.Interface, Size: 0},
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

// BuiltinKeywords are keywords built into Go -- for, range, etc
var BuiltinKeywords = []string{
	"break",
	"case",
	"chan",
	"const",
	"continue",
	"default",
	"defer",
	"else",
	"fallthrough",
	"for",
	"func",
	"go",
	"goto",
	"if",
	"import",
	"interface",
	"map",
	"package",
	"range",
	"return",
	"select",
	"struct",
	"switch",
	"type",
	"var",
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
