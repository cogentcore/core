// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mobile

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoSample is a part of the index of the Android SDK.
const repoSample = `<?xml version='1.0' encoding='utf-8'?>
<sdk:sdk-repository xmlns:sdk="http://schemas.android.com/sdk/android/repo/repository2/03" xmlns:common="http://schemas.android.com/repository/android/common/02" xmlns:generic="http://schemas.android.com/repository/android/generic/02" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <channel id="channel-0">stable</channel>
  <channel id="channel-1">beta</channel>
  <remotePackage path="ndk;28.2.13676358">
    <channelRef ref="channel-0"/>
  </remotePackage>
  <remotePackage path="ndk;29.0.14206865">
    <channelRef ref="channel-0"/>
  </remotePackage>
  <remotePackage path="ndk;30.0.1">
    <channelRef ref="channel-1"/>
  </remotePackage>
  <remotePackage path="ndk;28.10.1">
    <channelRef ref="channel-0"/>
  </remotePackage>
  <remotePackage path="cmdline-tools;latest">
    <channelRef ref="channel-0"/>
    <archives>
      <archive>
        <complete><size>1</size><url>commandlinetools-linux-1_latest.zip</url></complete>
        <host-os>linux</host-os>
      </archive>
      <archive>
        <complete><size>1</size><url>commandlinetools-mac_x86_64-1_latest.zip</url></complete>
        <host-os>macosx</host-os>
        <host-arch>x64</host-arch>
      </archive>
      <archive>
        <complete><size>1</size><url>commandlinetools-mac_arm64-1_latest.zip</url></complete>
        <host-os>macosx</host-os>
        <host-arch>aarch64</host-arch>
      </archive>
    </archives>
  </remotePackage>
</sdk:sdk-repository>`

// TestAndroidRepo tests reading the index of the Android SDK: the latest
// stable NDK, by the numbers of its version, and the archives of the
// command line tools for each system.
func TestAndroidRepo(t *testing.T) {
	pkgs, err := parseRepoPackages(strings.NewReader(repoSample))
	require.NoError(t, err)
	assert.Len(t, pkgs, 5)
	assert.Equal(t, "ndk;29.0.14206865", latestStable(pkgs, "ndk;"))
	tools := "cmdline-tools;latest"
	assert.Equal(t, androidRepo+"commandlinetools-mac_arm64-1_latest.zip", archiveURL(pkgs, tools, "darwin", "arm64"))
	assert.Equal(t, androidRepo+"commandlinetools-mac_x86_64-1_latest.zip", archiveURL(pkgs, tools, "darwin", "amd64"))
	assert.Equal(t, androidRepo+"commandlinetools-linux-1_latest.zip", archiveURL(pkgs, tools, "linux", "amd64"))
	assert.Equal(t, "", archiveURL(pkgs, tools, "windows", "amd64"))
}

// TestChooseAndroidDevice tests choosing the device to run apps on from
// the output of adb devices: a phone before an emulator, and only ready
// devices.
func TestChooseAndroidDevice(t *testing.T) {
	out := "List of devices attached\nemulator-5554\tdevice\nR5CT1234\tdevice\nX99\tunauthorized\n\n"
	devs := parseAndroidDevices(out)
	assert.Len(t, devs, 3)
	assert.Equal(t, "R5CT1234", chooseAndroidDevice(devs))
	assert.Equal(t, "emulator-5554", chooseAndroidDevice(parseAndroidDevices("List of devices attached\nemulator-5554\tdevice\nX99\tunauthorized\n")))
	assert.Equal(t, "", chooseAndroidDevice(parseAndroidDevices("List of devices attached\nX99\toffline\n")))
}

// TestAndroidSimDevice tests choosing the hardware profile of the AVD
// from those of avdmanager.
func TestAndroidSimDevice(t *testing.T) {
	assert.Equal(t, "pixel_6", androidSimDevice([]string{"pixel", "pixel_5", "pixel_6", "pixel_6_pro", "pixel_c"}))
	assert.Equal(t, "pixel_9", androidSimDevice([]string{"pixel_9", "pixel_8"}))
	assert.Equal(t, "pixel_xl", androidSimDevice([]string{"tv", "pixel_xl"}))
	assert.Equal(t, "tv", androidSimDevice([]string{"tv"}))
}

// TestSetIniValue tests setting the values of the config.ini of an AVD.
func TestSetIniValue(t *testing.T) {
	ini := "hw.gpu.enabled=yes\nhw.keyboard=no\nhw.keyboard.lid=yes\n"
	assert.Equal(t, "hw.gpu.enabled=yes\nhw.keyboard=yes\nhw.keyboard.lid=yes\n", setIniValue(ini, "hw.keyboard", "yes"))
	assert.Equal(t, "a = 1\nhw.keyboard=yes\n", setIniValue("a = 1\nhw.keyboard = no\n", "hw.keyboard", "yes"))
	assert.Equal(t, "a=1\nhw.keyboard=yes\n", setIniValue("a=1", "hw.keyboard", "yes"))
}
