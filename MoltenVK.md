# How to build and use a custom MoltenVK library

There are a lot of hoops to jump through, thanks to lots of Apple security things that result in things like the app just failing to run with `Killed: 9` and no further explanation.

* Download and install the latest SDK as a starting point from: https://vulkan.lunarg.com

* Follow all the README.md from https://github.com/KhronosGroup/MoltenVK -- in particular:
    + actually *start* XCode -- not enough to update it -- it does some updates when launched

* Example for how to use a modified dependency:
    + clone the dependency (e.g., https://github.com/KhronosGroup/SPIRV-Cross)
    + specify its path during build:
    
```bash
$ ./fetchDependencies --macos --spirv-cross-root /Users/oreilly/github/SPIRV-Cross
````

* `make macos` builds `.dylib` in: `Package/Release/MoltenVK/dylib/macOS/libMoltenVK.dylib`

* Ultimately, want to copy to `/usr/local/lib`, where the SDK puts its library:

```bash
$ sudo cp Package/Release/MoltenVK/dylib/macOS/libMoltenVK.dylib /usr/local/lib
```

* Also, there is this mysterious `.icd` file that needs to refer to the .dylib -- it won't load the library properly if you don't get this one right:

```bash
$ sudo cp Package/Release/MoltenVK/dylib/macOS/MoltenVK_icd.json /usr/local/share/vulkan/icd.d/MoltenVK_icd.json
```

Then edit that file so the `library_path` points to the full correct path:

```json
{
    "file_format_version" : "1.0.0",
    "ICD": {
        "library_path": "/usr/local/lib/libMoltenVK.dylib",
        "api_version" : "1.2.0",
        "is_portability_driver" : true
    }
}
```

* But if you just try to use it directly (`go build`, run), you get the `Killed: 9` error!

* Need to disable gatekeeper or else you'll have to reboot constantly:

```bash
$ sudo /usr/sbin/spctl --master-disable
```


* Metal library location is: `/Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/metal/macos/lib/clang/31001.667/include/metal`

* Compile a metal file directly to see what is going on:
    + do `make` in SPIRV-Cross top level, and copy `spirv-cross` exe to ~/bin

```bash
$ spirv-cross gpu_sendspike.spv --metal --msl-version 30000 >ss.metal
$ xcrun -sdk macosx metal ss.metal -std=metal3.0
```


I've got it working finally  -- it took me way too long to figure out that I just needed to disable gatekeeper to not have to reboot every iteration!  I tried a bunch of useless stuff about code signing certificates that never worked.

#if MVK_XCODE_14
	if ( mvkOSVersionIsAtLeast(12.0) ) {
		_metalFeatures.mslVersionEnum = MTLLanguageVersion3_0;
	}
#endif


# XCode developer certificates

Might be useful but definitely didn't help with rebooting -- disabling gatekeeper was the thing.

* Or sign the thing somehow: https://ioscodesigning.com -- in XCode, can sign in with apple id and get a certificate.
    + Xcode, Preferences, Accounts, + to add 
    + enter apple id
    + Do `Download Manual Profiles` -- doesn't work without this
    + Open `Keychain Access` app and find certificate, click on `Trust` and set to `Always Trust`
    + May need to reboot at this point.

Verify:
```bash
$ security find-identity -v -p codesigning
```    

Sign:
```bash
$ codesign -f -s rcoreilly@me.com Package/Release/MoltenVK/dylib/macOS/libMoltenVK.dylib
Warning: unable to build chain to self-signed root for signer "Apple Development: rcoreilly@me.com (86223M5MVQ)"
Package/Release/MoltenVK/dylib/macOS/libMoltenVK.dylib: errSecInternalComponent
```

This seems to fail, as confirmed by:
```bash
$ codesign -v Package/Release/MoltenVK/dylib/macOS/libMoltenVK.dylib
Package/Release/MoltenVK/dylib/macOS/libMoltenVK.dylib: code object is not signed at all
```
and:
```bash
$ /usr/sbin/spctl -a -t exec -vv Package/Release/MoltenVK/dylib/macOS/libMoltenVK.dylib
```

Bunch of stuff that didn't work: https://developer.apple.com/forums/thread/86161

Try installing: https://www.apple.com/certificateauthority/AppleWWDRCAG3.cer


