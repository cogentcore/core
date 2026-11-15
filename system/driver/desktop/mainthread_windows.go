// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import "golang.org/x/sys/windows"

// currentThreadID returns the id of the current thread.
func currentThreadID() int { return int(windows.GetCurrentThreadId()) }
