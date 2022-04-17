// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package egpu

func CheckExisting(actual, required []string) (existing []string, missing int) {
	existing = make([]string, 0, len(required))
	for j := range required {
		req := SafeString(required[j])
		for i := range actual {
			if SafeString(actual[i]) == req {
				existing = append(existing, req)
			}
		}
	}
	missing = len(required) - len(existing)
	return existing, missing
}

var end = "\x00"
var endChar byte = '\x00'

func SafeString(s string) string {
	if len(s) == 0 {
		return end
	}
	if s[len(s)-1] != endChar {
		return s + end
	}
	return s
}

func SafeStrings(list []string) []string {
	for i := range list {
		list[i] = SafeString(list[i])
	}
	return list
}
