// Copyright (c) 2023, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package econfig

import (
	"github.com/emer/emergent/elog"
	"github.com/emer/emergent/etime"
)

// LogFileName returns a standard log file name as netName_runName_logName.tsv
func LogFileName(logName, netName, runName string) string {
	return netName + "_" + runName + "_" + logName + ".tsv"
}

// SetLogFile sets the log file for given mode and time,
// using given logName (extension), netName and runName,
// if the Config flag is set.
func SetLogFile(logs *elog.Logs, configOn bool, mode etime.Modes, time etime.Times, logName, netName, runName string) {
	if !configOn {
		return
	}
	fnm := LogFileName(logName, netName, runName)
	logs.SetLogFile(mode, time, fnm)
}
