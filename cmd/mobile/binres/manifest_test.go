// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package binres

import (
	"strings"
	"testing"
)

// testManifest is a manifest as that of the apps that are built.
const testManifest = `
<manifest
	xmlns:android="http://schemas.android.com/apk/res/android"
	package="com.example.app"
	android:versionCode="1"
	android:versionName="1.0">
	<application android:label="App" android:debuggable="true">
	<activity android:name="org.golang.app.GoNativeActivity"
		android:label="App"
		android:configChanges="orientation|screenSize|keyboardHidden">
		<meta-data android:name="android.app.lib_name" android:value="App" />
		<intent-filter>
			<action android:name="android.intent.action.MAIN" />
			<category android:name="android.intent.category.LAUNCHER" />
		</intent-filter>
	</activity>
	</application>
</manifest>`

// TestManifestAttrOrder tests that the attributes of the elements of an
// encoded manifest are in the order of their resource IDs, which Android
// needs to find them, and that android:exported is added to the activity
// (once) for target SDK versions 31 and above, and only to it.
func TestManifestAttrOrder(t *testing.T) {
	skipSynthesize = false // (as when apps are built)
	defer func() { skipSynthesize = true }()
	bx, err := UnmarshalXML(strings.NewReader(testManifest), true, 23, 35)
	if err != nil {
		t.Fatal(err)
	}
	bin, err := bx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	dx := new(XML)
	if err := dx.UnmarshalBinary(bin); err != nil {
		t.Fatal(err)
	}
	exported := map[string]int{}
	var walk func(el *Element)
	walk = func(el *Element) {
		name := el.Name.Resolve(dx.Pool)
		last := uint32(0)
		for _, a := range el.attrs {
			an := a.Name.Resolve(dx.Pool)
			if an == "exported" {
				exported[name]++
			}
			id := uint32(0xFFFFFFFF)
			if a.NS != NoEntry && int(a.Name) < len(dx.Map.rs) {
				id = uint32(dx.Map.rs[a.Name])
			}
			if id < last {
				t.Errorf("%s: attribute %s (%#x) is after one with a larger resource ID (%#x)", name, an, id, last)
			}
			last = id
		}
		for _, c := range el.Children {
			walk(c)
		}
	}
	for _, el := range dx.Children {
		walk(el)
	}
	if exported["activity"] != 1 || len(exported) != 1 {
		t.Errorf("android:exported should be on the activity once, but is on %v", exported)
	}
}
