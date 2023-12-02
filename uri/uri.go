// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uri

import (
	"net/url"
	"strings"

	"goki.dev/icons"
)

// URI uniform resource identifier for general app resources.
type URI struct {

	// brief label
	Label string

	// URL path
	URL string

	// icon
	Icon icons.Icon

	// extra information, e.g. detailed description, type, arguments, etc;
	Desc string

	// lang specific or other, e.g. class or type
	Extra map[string]string

	// Function to call for this item, if not nil
	Func func()
}

func (ur URI) String() string {
	s := ""
	if ur.Label != "" {
		s += ur.Label + ": "
	}
	s += ur.URL
	return s
}

func (ur *URI) SetURL(scheme, host, path string) {
	// note: this is producing escaped URLs which is not user-friendly
	// I can't quite figure out how to unescape
	// u := url.URL{Scheme: scheme, Host: host, Path: path}
	// ur.URL = u.String()
	ur.URL = scheme + "://" + host + path
}

func (ur URI) HasScheme(scheme string) bool {
	u, err := url.Parse(ur.URL)
	if err == nil {
		if u.Scheme == scheme {
			return true
		}
	}
	return false
}

// URIs is a collection of URI
type URIs []URI

// todo: needs a lot smarter filtering
func (ur URIs) Filter(str string) URIs {
	sl := strings.ToLower(str)
	surl, suerr := url.Parse(sl)

	var fl URIs
	for _, u := range ur {
		if suerr == nil {
			lu := strings.ToLower(u.URL)
			turl, tuerr := url.Parse(lu)
			if tuerr == nil {
				if turl.Scheme != surl.Scheme {
					continue
				}
				if strings.Contains(turl.Host, surl.Host) || strings.Contains(turl.Path, surl.Path) {
					fl = append(fl, u)
					continue
				}
			}
		}
		tl := strings.ToLower(u.Label)
		if strings.Contains(tl, sl) {
			fl = append(fl, u)
			continue
		}
	}
	return fl
}
