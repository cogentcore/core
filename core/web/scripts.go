// Code generated by "gen/scripts.go"; DO NOT EDIT.

package web

const(
// DefaultAppWorkersJS is the default template used in [MakeAppWorkerJS] to generate app-worker.js.
DefaultAppWorkerJS = "const cacheName = \"app-\" + \"{{.Version}}\";\nconst resourcesToCache = {{.ResourcesToCache}};\n\nself.addEventListener(\"install\", (event) => {\n  console.log(\"installing app worker {{.Version}}\");\n\n  event.waitUntil(\n    caches\n      .open(cacheName)\n      .then((cache) => {\n        return cache.addAll(resourcesToCache);\n      })\n      .then(() => {\n        self.skipWaiting();\n      })\n  );\n});\n\nself.addEventListener(\"activate\", (event) => {\n  event.waitUntil(\n    caches.keys().then((keyList) => {\n      return Promise.all(\n        keyList.map((key) => {\n          if (key !== cacheName) {\n            return caches.delete(key);\n          }\n        })\n      );\n    })\n  );\n  console.log(\"app worker {{.Version}} is activated\");\n});\n\nself.addEventListener(\"fetch\", (event) => {\n  event.respondWith(\n    caches.match(event.request).then((response) => {\n      return response || fetch(event.request);\n    })\n  );\n});\n\nself.addEventListener(\"push\", (event) => {\n  if (!event.data || !event.data.text()) {\n    return;\n  }\n\n  const notification = JSON.parse(event.data.text());\n  if (!notification) {\n    return;\n  }\n\n  const title = notification.title;\n  delete notification.title;\n\n  if (!notification.data) {\n    notification.data = {};\n  }\n  let actions = [];\n  for (let i in notification.actions) {\n    const action = notification.actions[i];\n\n    actions.push({\n      action: action.action,\n      path: action.path,\n    });\n\n    delete action.path;\n  }\n  notification.data.goapp = {\n    path: notification.path,\n    actions: actions,\n  };\n  delete notification.path;\n\n  event.waitUntil(self.registration.showNotification(title, notification));\n});\n\nself.addEventListener(\"notificationclick\", (event) => {\n  event.notification.close();\n\n  const notification = event.notification;\n  let path = notification.data.goapp.path;\n\n  for (let i in notification.data.goapp.actions) {\n    const action = notification.data.goapp.actions[i];\n    if (action.action === event.action) {\n      path = action.path;\n      break;\n    }\n  }\n\n  event.waitUntil(\n    clients\n      .matchAll({\n        type: \"window\",\n      })\n      .then((clientList) => {\n        for (var i = 0; i < clientList.length; i++) {\n          let client = clientList[i];\n          if (\"focus\" in client) {\n            client.focus();\n            client.postMessage({\n              goapp: {\n                type: \"notification\",\n                path: path,\n              },\n            });\n            return;\n          }\n        }\n\n        if (clients.openWindow) {\n          return clients.openWindow(path);\n        }\n      })\n  );\n});\n"

// WASMExecJSGoCurrent is the wasm_exec.js file for the current version of Go.
WASMExecJSGoCurrent = "// Copyright 2018 The Go Authors. All rights reserved.\n// Use of this source code is governed by a BSD-style\n// license that can be found in the LICENSE file.\n\n\"use strict\";\n\n(() => {\n\tconst enosys = () => {\n\t\tconst err = new Error(\"not implemented\");\n\t\terr.code = \"ENOSYS\";\n\t\treturn err;\n\t};\n\n\tlet outputBuf = \"\";\n\tconst writeConsole = function (fd, buf) {\n\t\toutputBuf += decoder.decode(buf);\n\t\tconst nl = outputBuf.lastIndexOf(\"\\n\");\n\t\tif (nl != -1) {\n\t\t\tif (fd == 2) {\n\t\t\t\tconsole.error(outputBuf.substring(0, nl));\n\t\t\t} else {\n\t\t\t\tconsole.log(outputBuf.substring(0, nl));\n\t\t\t}\n\t\t\toutputBuf = outputBuf.substring(nl + 1);\n\t\t}\n\t\treturn buf.length;\n\t}\n\n\tif (!globalThis.fs) {\n\t\tglobalThis.fs = {\n\t\t\tconstants: { O_WRONLY: -1, O_RDWR: -1, O_CREAT: -1, O_TRUNC: -1, O_APPEND: -1, O_EXCL: -1 }, // temporary placeholder that is overwritten by jsfs\n\t\t};\n\t}\n\n\tif (!globalThis.process) {\n\t\tglobalThis.process = {\n\t\t\tgetuid() { return -1; },\n\t\t\tgetgid() { return -1; },\n\t\t\tgeteuid() { return -1; },\n\t\t\tgetegid() { return -1; },\n\t\t\tgetgroups() { throw enosys(); },\n\t\t\tpid: -1,\n\t\t\tppid: -1,\n\t\t\tumask() { throw enosys(); },\n\t\t\tcwd() { return \"/\" },\n\t\t\tchdir() { throw enosys(); },\n\t\t}\n\t}\n\n\tif (!globalThis.crypto) {\n\t\tthrow new Error(\"globalThis.crypto is not available, polyfill required (crypto.getRandomValues only)\");\n\t}\n\n\tif (!globalThis.performance) {\n\t\tthrow new Error(\"globalThis.performance is not available, polyfill required (performance.now only)\");\n\t}\n\n\tif (!globalThis.TextEncoder) {\n\t\tthrow new Error(\"globalThis.TextEncoder is not available, polyfill required\");\n\t}\n\n\tif (!globalThis.TextDecoder) {\n\t\tthrow new Error(\"globalThis.TextDecoder is not available, polyfill required\");\n\t}\n\n\tconst encoder = new TextEncoder(\"utf-8\");\n\tconst decoder = new TextDecoder(\"utf-8\");\n\n\tglobalThis.Go = class {\n\t\tconstructor() {\n\t\t\tthis.argv = [\"js\"];\n\t\t\tthis.env = {};\n\t\t\tthis.exit = (code) => {\n\t\t\t\tif (code !== 0) {\n\t\t\t\t\tconsole.error(\"exit code:\", code);\n\t\t\t\t}\n\t\t\t};\n\t\t\tthis._exitPromise = new Promise((resolve) => {\n\t\t\t\tthis._resolveExitPromise = resolve;\n\t\t\t});\n\t\t\tthis._pendingEvent = null;\n\t\t\tthis._scheduledTimeouts = new Map();\n\t\t\tthis._nextCallbackTimeoutID = 1;\n\n\t\t\tconst setInt64 = (addr, v) => {\n\t\t\t\tthis.mem.setUint32(addr + 0, v, true);\n\t\t\t\tthis.mem.setUint32(addr + 4, Math.floor(v / 4294967296), true);\n\t\t\t}\n\n\t\t\tconst setInt32 = (addr, v) => {\n\t\t\t\tthis.mem.setUint32(addr + 0, v, true);\n\t\t\t}\n\n\t\t\tconst getInt64 = (addr) => {\n\t\t\t\tconst low = this.mem.getUint32(addr + 0, true);\n\t\t\t\tconst high = this.mem.getInt32(addr + 4, true);\n\t\t\t\treturn low + high * 4294967296;\n\t\t\t}\n\n\t\t\tconst loadValue = (addr) => {\n\t\t\t\tconst f = this.mem.getFloat64(addr, true);\n\t\t\t\tif (f === 0) {\n\t\t\t\t\treturn undefined;\n\t\t\t\t}\n\t\t\t\tif (!isNaN(f)) {\n\t\t\t\t\treturn f;\n\t\t\t\t}\n\n\t\t\t\tconst id = this.mem.getUint32(addr, true);\n\t\t\t\treturn this._values[id];\n\t\t\t}\n\n\t\t\tconst storeValue = (addr, v) => {\n\t\t\t\tconst nanHead = 0x7FF80000;\n\n\t\t\t\tif (typeof v === \"number\" && v !== 0) {\n\t\t\t\t\tif (isNaN(v)) {\n\t\t\t\t\t\tthis.mem.setUint32(addr + 4, nanHead, true);\n\t\t\t\t\t\tthis.mem.setUint32(addr, 0, true);\n\t\t\t\t\t\treturn;\n\t\t\t\t\t}\n\t\t\t\t\tthis.mem.setFloat64(addr, v, true);\n\t\t\t\t\treturn;\n\t\t\t\t}\n\n\t\t\t\tif (v === undefined) {\n\t\t\t\t\tthis.mem.setFloat64(addr, 0, true);\n\t\t\t\t\treturn;\n\t\t\t\t}\n\n\t\t\t\tlet id = this._ids.get(v);\n\t\t\t\tif (id === undefined) {\n\t\t\t\t\tid = this._idPool.pop();\n\t\t\t\t\tif (id === undefined) {\n\t\t\t\t\t\tid = this._values.length;\n\t\t\t\t\t}\n\t\t\t\t\tthis._values[id] = v;\n\t\t\t\t\tthis._goRefCounts[id] = 0;\n\t\t\t\t\tthis._ids.set(v, id);\n\t\t\t\t}\n\t\t\t\tthis._goRefCounts[id]++;\n\t\t\t\tlet typeFlag = 0;\n\t\t\t\tswitch (typeof v) {\n\t\t\t\t\tcase \"object\":\n\t\t\t\t\t\tif (v !== null) {\n\t\t\t\t\t\t\ttypeFlag = 1;\n\t\t\t\t\t\t}\n\t\t\t\t\t\tbreak;\n\t\t\t\t\tcase \"string\":\n\t\t\t\t\t\ttypeFlag = 2;\n\t\t\t\t\t\tbreak;\n\t\t\t\t\tcase \"symbol\":\n\t\t\t\t\t\ttypeFlag = 3;\n\t\t\t\t\t\tbreak;\n\t\t\t\t\tcase \"function\":\n\t\t\t\t\t\ttypeFlag = 4;\n\t\t\t\t\t\tbreak;\n\t\t\t\t}\n\t\t\t\tthis.mem.setUint32(addr + 4, nanHead | typeFlag, true);\n\t\t\t\tthis.mem.setUint32(addr, id, true);\n\t\t\t}\n\n\t\t\tconst loadSlice = (addr) => {\n\t\t\t\tconst array = getInt64(addr + 0);\n\t\t\t\tconst len = getInt64(addr + 8);\n\t\t\t\treturn new Uint8Array(this._inst.exports.mem.buffer, array, len);\n\t\t\t}\n\n\t\t\tconst loadSliceOfValues = (addr) => {\n\t\t\t\tconst array = getInt64(addr + 0);\n\t\t\t\tconst len = getInt64(addr + 8);\n\t\t\t\tconst a = new Array(len);\n\t\t\t\tfor (let i = 0; i < len; i++) {\n\t\t\t\t\ta[i] = loadValue(array + i * 8);\n\t\t\t\t}\n\t\t\t\treturn a;\n\t\t\t}\n\n\t\t\tconst loadString = (addr) => {\n\t\t\t\tconst saddr = getInt64(addr + 0);\n\t\t\t\tconst len = getInt64(addr + 8);\n\t\t\t\treturn decoder.decode(new DataView(this._inst.exports.mem.buffer, saddr, len));\n\t\t\t}\n\n\t\t\tconst timeOrigin = Date.now() - performance.now();\n\t\t\tthis.importObject = {\n\t\t\t\t_gotest: {\n\t\t\t\t\tadd: (a, b) => a + b,\n\t\t\t\t},\n\t\t\t\tgojs: {\n\t\t\t\t\t// Go's SP does not change as long as no Go code is running. Some operations (e.g. calls, getters and setters)\n\t\t\t\t\t// may synchronously trigger a Go event handler. This makes Go code get executed in the middle of the imported\n\t\t\t\t\t// function. A goroutine can switch to a new stack if the current stack is too small (see morestack function).\n\t\t\t\t\t// This changes the SP, thus we have to update the SP used by the imported function.\n\n\t\t\t\t\t// func wasmExit(code int32)\n\t\t\t\t\t\"runtime.wasmExit\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tconst code = this.mem.getInt32(sp + 8, true);\n\t\t\t\t\t\tthis.exited = true;\n\t\t\t\t\t\tdelete this._inst;\n\t\t\t\t\t\tdelete this._values;\n\t\t\t\t\t\tdelete this._goRefCounts;\n\t\t\t\t\t\tdelete this._ids;\n\t\t\t\t\t\tdelete this._idPool;\n\t\t\t\t\t\tthis.exit(code);\n\t\t\t\t\t},\n\n\t\t\t\t\t// func wasmWrite(fd uintptr, p unsafe.Pointer, n int32)\n\t\t\t\t\t\"runtime.wasmWrite\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tconst fd = getInt64(sp + 8);\n\t\t\t\t\t\tconst p = getInt64(sp + 16);\n\t\t\t\t\t\tconst n = this.mem.getInt32(sp + 24, true);\n\t\t\t\t\t\twriteConsole(fd, new Uint8Array(this._inst.exports.mem.buffer, p, n));\n\t\t\t\t\t},\n\n\t\t\t\t\t// func resetMemoryDataView()\n\t\t\t\t\t\"runtime.resetMemoryDataView\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tthis.mem = new DataView(this._inst.exports.mem.buffer);\n\t\t\t\t\t},\n\n\t\t\t\t\t// func nanotime1() int64\n\t\t\t\t\t\"runtime.nanotime1\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tsetInt64(sp + 8, (timeOrigin + performance.now()) * 1000000);\n\t\t\t\t\t},\n\n\t\t\t\t\t// func walltime() (sec int64, nsec int32)\n\t\t\t\t\t\"runtime.walltime\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tconst msec = (new Date).getTime();\n\t\t\t\t\t\tsetInt64(sp + 8, msec / 1000);\n\t\t\t\t\t\tthis.mem.setInt32(sp + 16, (msec % 1000) * 1000000, true);\n\t\t\t\t\t},\n\n\t\t\t\t\t// func scheduleTimeoutEvent(delay int64) int32\n\t\t\t\t\t\"runtime.scheduleTimeoutEvent\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tconst id = this._nextCallbackTimeoutID;\n\t\t\t\t\t\tthis._nextCallbackTimeoutID++;\n\t\t\t\t\t\tthis._scheduledTimeouts.set(id, setTimeout(\n\t\t\t\t\t\t\t() => {\n\t\t\t\t\t\t\t\tthis._resume();\n\t\t\t\t\t\t\t\twhile (this._scheduledTimeouts.has(id)) {\n\t\t\t\t\t\t\t\t\t// for some reason Go failed to register the timeout event, log and try again\n\t\t\t\t\t\t\t\t\t// (temporary workaround for https://github.com/golang/go/issues/28975)\n\t\t\t\t\t\t\t\t\tconsole.warn(\"scheduleTimeoutEvent: missed timeout event\");\n\t\t\t\t\t\t\t\t\tthis._resume();\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t},\n\t\t\t\t\t\t\tgetInt64(sp + 8),\n\t\t\t\t\t\t));\n\t\t\t\t\t\tthis.mem.setInt32(sp + 16, id, true);\n\t\t\t\t\t},\n\n\t\t\t\t\t// func clearTimeoutEvent(id int32)\n\t\t\t\t\t\"runtime.clearTimeoutEvent\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tconst id = this.mem.getInt32(sp + 8, true);\n\t\t\t\t\t\tclearTimeout(this._scheduledTimeouts.get(id));\n\t\t\t\t\t\tthis._scheduledTimeouts.delete(id);\n\t\t\t\t\t},\n\n\t\t\t\t\t// func getRandomData(r []byte)\n\t\t\t\t\t\"runtime.getRandomData\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tcrypto.getRandomValues(loadSlice(sp + 8));\n\t\t\t\t\t},\n\n\t\t\t\t\t// func finalizeRef(v ref)\n\t\t\t\t\t\"syscall/js.finalizeRef\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tconst id = this.mem.getUint32(sp + 8, true);\n\t\t\t\t\t\tthis._goRefCounts[id]--;\n\t\t\t\t\t\tif (this._goRefCounts[id] === 0) {\n\t\t\t\t\t\t\tconst v = this._values[id];\n\t\t\t\t\t\t\tthis._values[id] = null;\n\t\t\t\t\t\t\tthis._ids.delete(v);\n\t\t\t\t\t\t\tthis._idPool.push(id);\n\t\t\t\t\t\t}\n\t\t\t\t\t},\n\n\t\t\t\t\t// func stringVal(value string) ref\n\t\t\t\t\t\"syscall/js.stringVal\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tstoreValue(sp + 24, loadString(sp + 8));\n\t\t\t\t\t},\n\n\t\t\t\t\t// func valueGet(v ref, p string) ref\n\t\t\t\t\t\"syscall/js.valueGet\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tconst result = Reflect.get(loadValue(sp + 8), loadString(sp + 16));\n\t\t\t\t\t\tsp = this._inst.exports.getsp() >>> 0; // see comment above\n\t\t\t\t\t\tstoreValue(sp + 32, result);\n\t\t\t\t\t},\n\n\t\t\t\t\t// func valueSet(v ref, p string, x ref)\n\t\t\t\t\t\"syscall/js.valueSet\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tReflect.set(loadValue(sp + 8), loadString(sp + 16), loadValue(sp + 32));\n\t\t\t\t\t},\n\n\t\t\t\t\t// func valueDelete(v ref, p string)\n\t\t\t\t\t\"syscall/js.valueDelete\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tReflect.deleteProperty(loadValue(sp + 8), loadString(sp + 16));\n\t\t\t\t\t},\n\n\t\t\t\t\t// func valueIndex(v ref, i int) ref\n\t\t\t\t\t\"syscall/js.valueIndex\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tstoreValue(sp + 24, Reflect.get(loadValue(sp + 8), getInt64(sp + 16)));\n\t\t\t\t\t},\n\n\t\t\t\t\t// valueSetIndex(v ref, i int, x ref)\n\t\t\t\t\t\"syscall/js.valueSetIndex\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tReflect.set(loadValue(sp + 8), getInt64(sp + 16), loadValue(sp + 24));\n\t\t\t\t\t},\n\n\t\t\t\t\t// func valueCall(v ref, m string, args []ref) (ref, bool)\n\t\t\t\t\t\"syscall/js.valueCall\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\ttry {\n\t\t\t\t\t\t\tconst v = loadValue(sp + 8);\n\t\t\t\t\t\t\tconst m = Reflect.get(v, loadString(sp + 16));\n\t\t\t\t\t\t\tconst args = loadSliceOfValues(sp + 32);\n\t\t\t\t\t\t\tconst result = Reflect.apply(m, v, args);\n\t\t\t\t\t\t\tsp = this._inst.exports.getsp() >>> 0; // see comment above\n\t\t\t\t\t\t\tstoreValue(sp + 56, result);\n\t\t\t\t\t\t\tthis.mem.setUint8(sp + 64, 1);\n\t\t\t\t\t\t} catch (err) {\n\t\t\t\t\t\t\tsp = this._inst.exports.getsp() >>> 0; // see comment above\n\t\t\t\t\t\t\tstoreValue(sp + 56, err);\n\t\t\t\t\t\t\tthis.mem.setUint8(sp + 64, 0);\n\t\t\t\t\t\t}\n\t\t\t\t\t},\n\n\t\t\t\t\t// func valueInvoke(v ref, args []ref) (ref, bool)\n\t\t\t\t\t\"syscall/js.valueInvoke\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\ttry {\n\t\t\t\t\t\t\tconst v = loadValue(sp + 8);\n\t\t\t\t\t\t\tconst args = loadSliceOfValues(sp + 16);\n\t\t\t\t\t\t\tconst result = Reflect.apply(v, undefined, args);\n\t\t\t\t\t\t\tsp = this._inst.exports.getsp() >>> 0; // see comment above\n\t\t\t\t\t\t\tstoreValue(sp + 40, result);\n\t\t\t\t\t\t\tthis.mem.setUint8(sp + 48, 1);\n\t\t\t\t\t\t} catch (err) {\n\t\t\t\t\t\t\tsp = this._inst.exports.getsp() >>> 0; // see comment above\n\t\t\t\t\t\t\tstoreValue(sp + 40, err);\n\t\t\t\t\t\t\tthis.mem.setUint8(sp + 48, 0);\n\t\t\t\t\t\t}\n\t\t\t\t\t},\n\n\t\t\t\t\t// func valueNew(v ref, args []ref) (ref, bool)\n\t\t\t\t\t\"syscall/js.valueNew\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\ttry {\n\t\t\t\t\t\t\tconst v = loadValue(sp + 8);\n\t\t\t\t\t\t\tconst args = loadSliceOfValues(sp + 16);\n\t\t\t\t\t\t\tconst result = Reflect.construct(v, args);\n\t\t\t\t\t\t\tsp = this._inst.exports.getsp() >>> 0; // see comment above\n\t\t\t\t\t\t\tstoreValue(sp + 40, result);\n\t\t\t\t\t\t\tthis.mem.setUint8(sp + 48, 1);\n\t\t\t\t\t\t} catch (err) {\n\t\t\t\t\t\t\tsp = this._inst.exports.getsp() >>> 0; // see comment above\n\t\t\t\t\t\t\tstoreValue(sp + 40, err);\n\t\t\t\t\t\t\tthis.mem.setUint8(sp + 48, 0);\n\t\t\t\t\t\t}\n\t\t\t\t\t},\n\n\t\t\t\t\t// func valueLength(v ref) int\n\t\t\t\t\t\"syscall/js.valueLength\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tsetInt64(sp + 16, parseInt(loadValue(sp + 8).length));\n\t\t\t\t\t},\n\n\t\t\t\t\t// valuePrepareString(v ref) (ref, int)\n\t\t\t\t\t\"syscall/js.valuePrepareString\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tconst str = encoder.encode(String(loadValue(sp + 8)));\n\t\t\t\t\t\tstoreValue(sp + 16, str);\n\t\t\t\t\t\tsetInt64(sp + 24, str.length);\n\t\t\t\t\t},\n\n\t\t\t\t\t// valueLoadString(v ref, b []byte)\n\t\t\t\t\t\"syscall/js.valueLoadString\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tconst str = loadValue(sp + 8);\n\t\t\t\t\t\tloadSlice(sp + 16).set(str);\n\t\t\t\t\t},\n\n\t\t\t\t\t// func valueInstanceOf(v ref, t ref) bool\n\t\t\t\t\t\"syscall/js.valueInstanceOf\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tthis.mem.setUint8(sp + 24, (loadValue(sp + 8) instanceof loadValue(sp + 16)) ? 1 : 0);\n\t\t\t\t\t},\n\n\t\t\t\t\t// func copyBytesToGo(dst []byte, src ref) (int, bool)\n\t\t\t\t\t\"syscall/js.copyBytesToGo\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tconst dst = loadSlice(sp + 8);\n\t\t\t\t\t\tconst src = loadValue(sp + 32);\n\t\t\t\t\t\tif (!(src instanceof Uint8Array || src instanceof Uint8ClampedArray)) {\n\t\t\t\t\t\t\tthis.mem.setUint8(sp + 48, 0);\n\t\t\t\t\t\t\treturn;\n\t\t\t\t\t\t}\n\t\t\t\t\t\tconst toCopy = src.subarray(0, dst.length);\n\t\t\t\t\t\tdst.set(toCopy);\n\t\t\t\t\t\tsetInt64(sp + 40, toCopy.length);\n\t\t\t\t\t\tthis.mem.setUint8(sp + 48, 1);\n\t\t\t\t\t},\n\n\t\t\t\t\t// func copyBytesToJS(dst ref, src []byte) (int, bool)\n\t\t\t\t\t\"syscall/js.copyBytesToJS\": (sp) => {\n\t\t\t\t\t\tsp >>>= 0;\n\t\t\t\t\t\tconst dst = loadValue(sp + 8);\n\t\t\t\t\t\tconst src = loadSlice(sp + 16);\n\t\t\t\t\t\tif (!(dst instanceof Uint8Array || dst instanceof Uint8ClampedArray)) {\n\t\t\t\t\t\t\tthis.mem.setUint8(sp + 48, 0);\n\t\t\t\t\t\t\treturn;\n\t\t\t\t\t\t}\n\t\t\t\t\t\tconst toCopy = src.subarray(0, dst.length);\n\t\t\t\t\t\tdst.set(toCopy);\n\t\t\t\t\t\tsetInt64(sp + 40, toCopy.length);\n\t\t\t\t\t\tthis.mem.setUint8(sp + 48, 1);\n\t\t\t\t\t},\n\n\t\t\t\t\t\"debug\": (value) => {\n\t\t\t\t\t\tconsole.log(value);\n\t\t\t\t\t},\n\t\t\t\t}\n\t\t\t};\n\t\t}\n\n\t\tasync run(instance) {\n\t\t\tif (!(instance instanceof WebAssembly.Instance)) {\n\t\t\t\tthrow new Error(\"Go.run: WebAssembly.Instance expected\");\n\t\t\t}\n\t\t\tthis._inst = instance;\n\t\t\tthis.mem = new DataView(this._inst.exports.mem.buffer);\n\t\t\tthis._values = [ // JS values that Go currently has references to, indexed by reference id\n\t\t\t\tNaN,\n\t\t\t\t0,\n\t\t\t\tnull,\n\t\t\t\ttrue,\n\t\t\t\tfalse,\n\t\t\t\tglobalThis,\n\t\t\t\tthis,\n\t\t\t];\n\t\t\tthis._goRefCounts = new Array(this._values.length).fill(Infinity); // number of references that Go has to a JS value, indexed by reference id\n\t\t\tthis._ids = new Map([ // mapping from JS values to reference ids\n\t\t\t\t[0, 1],\n\t\t\t\t[null, 2],\n\t\t\t\t[true, 3],\n\t\t\t\t[false, 4],\n\t\t\t\t[globalThis, 5],\n\t\t\t\t[this, 6],\n\t\t\t]);\n\t\t\tthis._idPool = [];   // unused ids that have been garbage collected\n\t\t\tthis.exited = false; // whether the Go program has exited\n\n\t\t\t// Pass command line arguments and environment variables to WebAssembly by writing them to the linear memory.\n\t\t\tlet offset = 4096;\n\n\t\t\tconst strPtr = (str) => {\n\t\t\t\tconst ptr = offset;\n\t\t\t\tconst bytes = encoder.encode(str + \"\\0\");\n\t\t\t\tnew Uint8Array(this.mem.buffer, offset, bytes.length).set(bytes);\n\t\t\t\toffset += bytes.length;\n\t\t\t\tif (offset % 8 !== 0) {\n\t\t\t\t\toffset += 8 - (offset % 8);\n\t\t\t\t}\n\t\t\t\treturn ptr;\n\t\t\t};\n\n\t\t\tconst argc = this.argv.length;\n\n\t\t\tconst argvPtrs = [];\n\t\t\tthis.argv.forEach((arg) => {\n\t\t\t\targvPtrs.push(strPtr(arg));\n\t\t\t});\n\t\t\targvPtrs.push(0);\n\n\t\t\tconst keys = Object.keys(this.env).sort();\n\t\t\tkeys.forEach((key) => {\n\t\t\t\targvPtrs.push(strPtr(`${key}=${this.env[key]}`));\n\t\t\t});\n\t\t\targvPtrs.push(0);\n\n\t\t\tconst argv = offset;\n\t\t\targvPtrs.forEach((ptr) => {\n\t\t\t\tthis.mem.setUint32(offset, ptr, true);\n\t\t\t\tthis.mem.setUint32(offset + 4, 0, true);\n\t\t\t\toffset += 8;\n\t\t\t});\n\n\t\t\t// The linker guarantees global data starts from at least wasmMinDataAddr.\n\t\t\t// Keep in sync with cmd/link/internal/ld/data.go:wasmMinDataAddr.\n\t\t\tconst wasmMinDataAddr = 4096 + 8192;\n\t\t\tif (offset >= wasmMinDataAddr) {\n\t\t\t\tthrow new Error(\"total length of command line and environment variables exceeds limit\");\n\t\t\t}\n\n\t\t\tthis._inst.exports.run(argc, argv);\n\t\t\tif (this.exited) {\n\t\t\t\tthis._resolveExitPromise();\n\t\t\t}\n\t\t\tawait this._exitPromise;\n\t\t}\n\n\t\t_resume() {\n\t\t\tif (this.exited) {\n\t\t\t\tthrow new Error(\"Go program has already exited\");\n\t\t\t}\n\t\t\tthis._inst.exports.resume();\n\t\t\tif (this.exited) {\n\t\t\t\tthis._resolveExitPromise();\n\t\t\t}\n\t\t}\n\n\t\t_makeFuncWrapper(id) {\n\t\t\tconst go = this;\n\t\t\treturn function () {\n\t\t\t\tconst event = { id: id, this: this, args: arguments };\n\t\t\t\tgo._pendingEvent = event;\n\t\t\t\tgo._resume();\n\t\t\t\treturn event.result;\n\t\t\t};\n\t\t}\n\t}\n})();\n"

// AppJS is the string used for [AppJSTmpl].
AppJS = "// -----------------------------------------------------------------------------\n// go-app\n// -----------------------------------------------------------------------------\nvar goappNav = function () { };\nvar goappOnUpdate = function () { };\nvar goappOnAppInstallChange = function () { };\n\nconst goappEnv = {{.Env }};\nconst goappLoadingLabel = \"{{.LoadingLabel}}\";\nconst goappWasmContentLengthHeader = \"{{.WasmContentLengthHeader}}\";\n\nlet wasm;\nlet memoryBytes;\n\nlet goappServiceWorkerRegistration;\nlet deferredPrompt = null;\n\ngoappInitServiceWorker();\ngoappWatchForUpdate();\ngoappWatchForInstallable();\ngoappInitWebAssembly();\n\n// -----------------------------------------------------------------------------\n// Service Worker\n// -----------------------------------------------------------------------------\nasync function goappInitServiceWorker() {\n  if (\"serviceWorker\" in navigator) {\n    try {\n      const registration = await navigator.serviceWorker.register(\n        \"{{.WorkerJS}}\"\n      );\n\n      goappServiceWorkerRegistration = registration;\n      goappSetupNotifyUpdate(registration);\n      goappSetupAutoUpdate(registration);\n      goappSetupPushNotification();\n    } catch (err) {\n      console.error(\"goapp service worker registration failed\", err);\n    }\n  }\n}\n\n// -----------------------------------------------------------------------------\n// Update\n// -----------------------------------------------------------------------------\nfunction goappWatchForUpdate() {\n  window.addEventListener(\"beforeinstallprompt\", (e) => {\n    e.preventDefault();\n    deferredPrompt = e;\n    goappOnAppInstallChange();\n  });\n}\n\nfunction goappSetupNotifyUpdate(registration) {\n  registration.addEventListener(\"updatefound\", (event) => {\n    const newSW = registration.installing;\n    newSW.addEventListener(\"statechange\", (event) => {\n      if (!navigator.serviceWorker.controller) {\n        return;\n      }\n      if (newSW.state != \"installed\") {\n        return;\n      }\n      goappOnUpdate();\n    });\n  });\n}\n\nfunction goappSetupAutoUpdate(registration) {\n  const autoUpdateInterval = \"{{.AutoUpdateInterval}}\";\n  if (autoUpdateInterval == 0) {\n    return;\n  }\n\n  window.setInterval(() => {\n    registration.update();\n  }, autoUpdateInterval);\n}\n\n// -----------------------------------------------------------------------------\n// Install\n// -----------------------------------------------------------------------------\nfunction goappWatchForInstallable() {\n  window.addEventListener(\"appinstalled\", () => {\n    deferredPrompt = null;\n    goappOnAppInstallChange();\n  });\n}\n\nfunction goappIsAppInstallable() {\n  return !goappIsAppInstalled() && deferredPrompt != null;\n}\n\nfunction goappIsAppInstalled() {\n  const isStandalone = window.matchMedia(\"(display-mode: standalone)\").matches;\n  return isStandalone || navigator.standalone;\n}\n\nasync function goappShowInstallPrompt() {\n  deferredPrompt.prompt();\n  await deferredPrompt.userChoice;\n  deferredPrompt = null;\n}\n\n// -----------------------------------------------------------------------------\n// Environment\n// -----------------------------------------------------------------------------\nfunction goappGetenv(k) {\n  return goappEnv[k];\n}\n\n// -----------------------------------------------------------------------------\n// Notifications\n// -----------------------------------------------------------------------------\nfunction goappSetupPushNotification() {\n  navigator.serviceWorker.addEventListener(\"message\", (event) => {\n    const msg = event.data.goapp;\n    if (!msg) {\n      return;\n    }\n\n    if (msg.type !== \"notification\") {\n      return;\n    }\n\n    goappNav(msg.path);\n  });\n}\n\nasync function goappSubscribePushNotifications(vapIDpublicKey) {\n  try {\n    const subscription =\n      await goappServiceWorkerRegistration.pushManager.subscribe({\n        userVisibleOnly: true,\n        applicationServerKey: vapIDpublicKey,\n      });\n    return JSON.stringify(subscription);\n  } catch (err) {\n    console.error(err);\n    return \"\";\n  }\n}\n\nfunction goappNewNotification(jsonNotification) {\n  let notification = JSON.parse(jsonNotification);\n\n  const title = notification.title;\n  delete notification.title;\n\n  let path = notification.path;\n  if (!path) {\n    path = \"/\";\n  }\n\n  const webNotification = new Notification(title, notification);\n\n  webNotification.onclick = () => {\n    goappNav(path);\n    webNotification.close();\n  };\n}\n\n// -----------------------------------------------------------------------------\n// Display Image\n// -----------------------------------------------------------------------------\n\nconst appCanvas = document.getElementById('app');\nconst appCanvasCtx = appCanvas.getContext('2d');\n\n// displayImage takes the pointer to the target image in the wasm linear memory\n// and its length. Then, it gets the resulting byte slice and creates an image data\n// with the given width and height.\nfunction displayImage(pointer, length, w, h) {\n  // if it doesn't exist or is detached, we have to make it\n  if (!memoryBytes || memoryBytes.byteLength === 0) {\n    memoryBytes = new Uint8ClampedArray(wasm.instance.exports.mem.buffer);\n  }\n\n  // using subarray instead of slice gives a 5x performance improvement due to no copying\n  let bytes = memoryBytes.subarray(pointer, pointer + length);\n  let data = new ImageData(bytes, w, h);\n  appCanvasCtx.putImageData(data, 0, 0);\n}\n\n// -----------------------------------------------------------------------------\n// Keep Clean Body\n// -----------------------------------------------------------------------------\nfunction goappKeepBodyClean() {\n  const body = document.body;\n  const bodyChildrenCount = body.children.length;\n\n  const mutationObserver = new MutationObserver(function (mutationList) {\n    mutationList.forEach((mutation) => {\n      switch (mutation.type) {\n        case \"childList\":\n          while (body.children.length > bodyChildrenCount) {\n            body.removeChild(body.lastChild);\n          }\n          break;\n      }\n    });\n  });\n\n  mutationObserver.observe(document.body, {\n    childList: true,\n  });\n\n  return () => mutationObserver.disconnect();\n}\n\n// -----------------------------------------------------------------------------\n// Web Assembly\n// -----------------------------------------------------------------------------\n\nasync function goappInitWebAssembly() {\n  const loader = document.getElementById(\"app-wasm-loader\");\n\n  if (!goappCanLoadWebAssembly()) {\n    loader.remove();\n    return;\n  }\n\n  let instantiateStreaming = WebAssembly.instantiateStreaming;\n  if (!instantiateStreaming) {\n    instantiateStreaming = async (resp, importObject) => {\n      const source = await (await resp).arrayBuffer();\n      // memoryBytes = new Uint8Array(resp.instance.exports.mem.buffer);\n      // console.log(\"got memory bytes\", memoryBytes);\n      return await WebAssembly.instantiate(source, importObject);\n    };\n  }\n\n  const loaderIcon = document.getElementById(\"app-wasm-loader-icon\");\n  const loaderLabel = document.getElementById(\"app-wasm-loader-label\");\n\n  try {\n    const showProgress = (progress) => {\n      loaderLabel.innerText = goappLoadingLabel.replace(\"{progress}\", progress);\n    };\n    showProgress(0);\n\n    const go = new Go();\n    wasm = await instantiateStreaming(\n      fetchWithProgress(\"{{.Wasm}}\", showProgress),\n      go.importObject,\n    );\n    go.run(wasm.instance);\n    loader.remove();\n  } catch (err) {\n    loaderIcon.className = \"goapp-logo\";\n    loaderLabel.innerText = err;\n    console.error(\"loading wasm failed: \", err);\n  }\n}\n\nfunction goappCanLoadWebAssembly() {\n  if (\n    /bot|googlebot|crawler|spider|robot|crawling/i.test(navigator.userAgent)\n  ) {\n    return false;\n  }\n\n  const urlParams = new URLSearchParams(window.location.search);\n  return urlParams.get(\"wasm\") !== \"false\";\n}\n\nasync function fetchWithProgress(url, progess) {\n  const response = await fetch(url);\n\n  let contentLength;\n  try {\n    contentLength = response.headers.get(goappWasmContentLengthHeader);\n  } catch { }\n  if (!goappWasmContentLengthHeader || !contentLength) {\n    contentLength = response.headers.get(\"Content-Length\");\n  }\n  if (!contentLength) {\n    contentLength = \"60000000\"; // 60 mb default\n  }\n\n  const total = parseInt(contentLength, 10);\n  let loaded = 0;\n\n  const progressHandler = function (loaded, total) {\n    progess(Math.round((loaded * 100) / total));\n  };\n\n  var res = new Response(\n    new ReadableStream(\n      {\n        async start(controller) {\n          var reader = response.body.getReader();\n          for (; ;) {\n            var { done, value } = await reader.read();\n\n            if (done) {\n              progressHandler(total, total);\n              break;\n            }\n\n            loaded += value.byteLength;\n            progressHandler(loaded, total);\n            controller.enqueue(value);\n          }\n          controller.close();\n        },\n      },\n      {\n        status: response.status,\n        statusText: response.statusText,\n      }\n    )\n  );\n\n  for (var pair of response.headers.entries()) {\n    res.headers.set(pair[0], pair[1]);\n  }\n\n  return res;\n}\n"

// ManifestJSON is the string used for [ManifestJSONTmpl].
ManifestJSON = "{\n  \"short_name\": \"{{.ShortName}}\",\n  \"name\": \"{{.Name}}\",\n  \"description\": \"{{.Description}}\",\n  \"icons\": [\n    {\n      \"src\": \"{{.SVGIcon}}\",\n      \"type\": \"image/svg+xml\",\n      \"sizes\": \"any\"\n    },\n    {\n      \"src\": \"{{.LargeIcon}}\",\n      \"type\": \"image/png\",\n      \"sizes\": \"512x512\"\n    },\n    {\n      \"src\": \"{{.DefaultIcon}}\",\n      \"type\": \"image/png\",\n      \"sizes\": \"192x192\"\n    }\n  ],\n  \"scope\": \"{{.Scope}}\",\n  \"start_url\": \"{{.StartURL}}\",\n  \"background_color\": \"{{.BackgroundColor}}\",\n  \"theme_color\": \"{{.ThemeColor}}\",\n  \"display\": \"standalone\"\n}"

// AppCSS is the string used for app.css.
AppCSS = "body {\n  margin: 0;\n  overflow: hidden;\n}\n\n#app {\n  width: 100vw;\n  height: 100vh;\n\n  /* no selection of canvas */\n  -webkit-touch-callout: none;\n  -webkit-user-select: none;\n  -khtml-user-select: none;\n  -moz-user-select: none;\n  -ms-user-select: none;\n  user-select: none;\n  outline: none;\n  -webkit-tap-highlight-color: rgba(255, 255, 255, 0); /* mobile webkit */\n}\n\n#text-field {\n  position: fixed;\n  opacity: 0;\n  top: 0;\n  left: 0;\n}\n\n/*------------------------------------------------------------------------------\n  Loader\n------------------------------------------------------------------------------*/\n.goapp-app-info {\n  position: fixed;\n  top: 0;\n  left: 0;\n  z-index: 1000;\n  width: 100vw;\n  height: 100vh;\n  overflow: hidden;\n\n  display: flex;\n  flex-direction: column;\n  justify-content: center;\n  align-items: center;\n\n  font-family: -apple-system, BlinkMacSystemFont, \"Segoe UI\", Roboto, Oxygen,\n    Ubuntu, Cantarell, \"Open Sans\", \"Helvetica Neue\", sans-serif;\n  font-size: 13px;\n  font-weight: 400;\n  color: white;\n  background-color: #2d2c2c;\n}\n\n@media (prefers-color-scheme: light) {\n  .goapp-app-info {\n    color: black;\n    background-color: #f6f6f6;\n  }\n}\n\n.goapp-logo {\n  max-width: 100px;\n  max-height: 100px;\n  user-select: none;\n  -moz-user-select: none;\n  -webkit-user-drag: none;\n  -webkit-user-select: none;\n  -ms-user-select: none;\n}\n\n.goapp-label {\n  margin-top: 12px;\n  font-size: 21px;\n  font-weight: 100;\n  letter-spacing: 1px;\n  max-width: 480px;\n  text-align: center;\n}\n\n.goapp-spin {\n  animation: goapp-spin-frames 1.21s infinite linear;\n}\n\n@keyframes goapp-spin-frames {\n  from {\n    transform: rotate(0deg);\n  }\n\n  to {\n    transform: rotate(360deg);\n  }\n}\n\n/*------------------------------------------------------------------------------\n  Not found\n------------------------------------------------------------------------------*/\n.goapp-notfound-title {\n  display: flex;\n  justify-content: center;\n  align-items: center;\n  font-size: 65pt;\n  font-weight: 100;\n}\n"

// IndexHTML is the string used for [IndexHTMLTmpl].
IndexHTML = "<!DOCTYPE html>\n<html lang=\"en\">\n\n<head>\n    <meta charset=\"UTF-8\">\n    <meta name=\"author\" content=\"{{.Author}}\">\n    <meta name=\"description\" content=\"{{.Desc}}\">\n    <meta name=\"keywords\" content=\"{{.Keywords}}\">\n    <meta name=\"theme-color\" content=\"{{.ThemeColor}}\">\n    <meta name=\"viewport\"\n        content=\"width=device-width, initial-scale=1, maximum-scale=1, user-scalable=0, viewport-fit=cover\">\n    <meta property=\"og:url\" content=\"http://127.0.0.1:60452/\">\n    <meta property=\"og:title\" content=\"{{.Title}}\">\n    <meta property=\"og:description\" content=\"{{.Desc}}\">\n    <meta property=\"og:type\" content=\"website\">\n    <meta property=\"og:image\" content=\"{{.Image}}\">\n    <title>{{.Title}}</title>\n    {{- if .VanityURL -}}\n    <meta name=\"go-import\" content=\"{{.VanityURL}} git https://github.com/{{.GithubVanityRepository}}\">\n    <meta name=\"go-source\"\n        content=\"{{.VanityURL}} https://github.com/{{.GithubVanityRepository}} https://github.com/{{.GithubVanityRepository}}/tree/master{/dir} https://github.com/{{.GithubVanityRepository}}/blob/master{/dir}/{file}#L{line}\">\n    {{- end -}}\n    <link type=\"text/css\" rel=\"preload\" href=\"app.css\" as=\"style\">\n    <link rel=\"icon\" href=\"icons/svg.svg\">\n    <link rel=\"apple-touch-icon\" href=\"icons/192.png\">\n    <link rel=\"manifest\" href=\"manifest.webmanifest\">\n    <link rel=\"stylesheet\" type=\"text/css\" href=\"app.css\">\n    <script defer src=\"wasm_exec.js\"></script>\n    <script defer src=\"app.js\"></script>\n</head>\n\n<body>\n    <canvas id=\"app\" contenteditable=\"true\"></canvas>\n    <input id=\"text-field\" type=\"text\">\n    <aside id=\"app-wasm-loader\" class=\"goapp-app-info\">\n        <img class=\"goapp-logo goapp-spin\" src=\"icons/512.png\" id=\"app-wasm-loader-icon\">\n        <p id=\"app-wasm-loader-label\" class=\"goapp-label\">Loading {{.Title}}...</p>\n    </aside>\n</body>\n\n</html>"

)
