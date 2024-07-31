// -----------------------------------------------------------------------------
// app
// -----------------------------------------------------------------------------
var appNav = function () { };
var appOnUpdate = function () { };
var appOnAppInstallChange = function () { };

const appEnv = {{.Env }};
const appWasmContentLengthHeader = "{{.WasmContentLengthHeader}}";

let wasm;

let appServiceWorkerRegistration;
let deferredPrompt = null;

appInitServiceWorker();
appWatchForUpdate();
appWatchForInstallable();
appInitWebAssembly();

// -----------------------------------------------------------------------------
// Service Worker
// -----------------------------------------------------------------------------
async function appInitServiceWorker() {
  if ("serviceWorker" in navigator) {
    try {
      const registration = await navigator.serviceWorker.register(
        document.documentElement.dataset.basePath + "app-worker.js"
      );

      appServiceWorkerRegistration = registration;
      appSetupNotifyUpdate(registration);
      appSetupAutoUpdate(registration);
      appSetupPushNotification();
    } catch (err) {
      console.error("app service worker registration failed", err);
    }
  }
}

// -----------------------------------------------------------------------------
// Update
// -----------------------------------------------------------------------------
function appWatchForUpdate() {
  window.addEventListener("beforeinstallprompt", (e) => {
    e.preventDefault();
    deferredPrompt = e;
    appOnAppInstallChange();
  });
}

function appSetupNotifyUpdate(registration) {
  registration.addEventListener("updatefound", (event) => {
    const newSW = registration.installing;
    newSW.addEventListener("statechange", (event) => {
      if (!navigator.serviceWorker.controller) {
        return;
      }
      if (newSW.state != "installed") {
        return;
      }
      appOnUpdate();
    });
  });
}

function appSetupAutoUpdate(registration) {
  const autoUpdateInterval = "{{.AutoUpdateInterval}}";
  if (autoUpdateInterval == 0) {
    return;
  }

  window.setInterval(() => {
    registration.update();
  }, autoUpdateInterval);
}

// -----------------------------------------------------------------------------
// Install
// -----------------------------------------------------------------------------
function appWatchForInstallable() {
  window.addEventListener("appinstalled", () => {
    deferredPrompt = null;
    appOnAppInstallChange();
  });
}

function appIsAppInstallable() {
  return !appIsAppInstalled() && deferredPrompt != null;
}

function appIsAppInstalled() {
  const isStandalone = window.matchMedia("(display-mode: standalone)").matches;
  return isStandalone || navigator.standalone;
}

async function appShowInstallPrompt() {
  deferredPrompt.prompt();
  await deferredPrompt.userChoice;
  deferredPrompt = null;
}

// -----------------------------------------------------------------------------
// Environment
// -----------------------------------------------------------------------------
function appGetenv(k) {
  return appEnv[k];
}

// -----------------------------------------------------------------------------
// Notifications
// -----------------------------------------------------------------------------
function appSetupPushNotification() {
  navigator.serviceWorker.addEventListener("message", (event) => {
    const msg = event.data.app;
    if (!msg) {
      return;
    }

    if (msg.type !== "notification") {
      return;
    }

    appNav(msg.path);
  });
}

async function appSubscribePushNotifications(vapIDpublicKey) {
  try {
    const subscription =
      await appServiceWorkerRegistration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: vapIDpublicKey,
      });
    return JSON.stringify(subscription);
  } catch (err) {
    console.error(err);
    return "";
  }
}

function appNewNotification(jsonNotification) {
  let notification = JSON.parse(jsonNotification);

  const title = notification.title;
  delete notification.title;

  let path = notification.path;
  if (!path) {
    path = "/";
  }

  const webNotification = new Notification(title, notification);

  webNotification.onclick = () => {
    appNav(path);
    webNotification.close();
  };
}

// -----------------------------------------------------------------------------
// Display Image
// -----------------------------------------------------------------------------

const appCanvas = document.getElementById('app');
const appCanvasCtx = appCanvas.getContext('2d');

// displayImage takes the pointer to the target image in the wasm linear memory
// and its length. Then, it gets the resulting byte slice and creates an image data
// with the given width and height.
function displayImage(pointer, length, w, h) {
  const memoryBytes = new Uint8ClampedArray(wasm.instance.exports.mem.buffer);

  // using subarray instead of slice gives a 5x performance improvement due to no copying
  const bytes = memoryBytes.subarray(pointer, pointer + length);
  const data = new ImageData(bytes, w, h);
  appCanvasCtx.putImageData(data, 0, 0);
}

// -----------------------------------------------------------------------------
// Keep Clean Body
// -----------------------------------------------------------------------------
function appKeepBodyClean() {
  const body = document.body;
  const bodyChildrenCount = body.children.length;

  const mutationObserver = new MutationObserver(function (mutationList) {
    mutationList.forEach((mutation) => {
      switch (mutation.type) {
        case "childList":
          while (body.children.length > bodyChildrenCount) {
            body.removeChild(body.lastChild);
          }
          break;
      }
    });
  });

  mutationObserver.observe(document.body, {
    childList: true,
  });

  return () => mutationObserver.disconnect();
}

// -----------------------------------------------------------------------------
// Web Assembly
// -----------------------------------------------------------------------------

async function appInitWebAssembly() {
  const loader = document.getElementById("app-wasm-loader");

  if (!appCanLoadWebAssembly()) {
    loader.remove();
    return;
  }

  let instantiateStreaming = WebAssembly.instantiateStreaming;
  if (!instantiateStreaming) {
    instantiateStreaming = async (resp, importObject) => {
      const source = await (await resp).arrayBuffer();
      return await WebAssembly.instantiate(source, importObject);
    };
  }

  const loaderLabel = document.getElementById("app-wasm-loader-label");
  const loaderProgress = document.getElementById("app-wasm-loader-progress");

  try {
    const showProgress = (progress) => {
      loaderLabel.innerText = progress + "%";
      loaderProgress.value = progress / 100;
    };
    showProgress(1);

    const go = new Go();
    wasm = await instantiateStreaming(
      fetchWithProgress(document.documentElement.dataset.basePath + "app.wasm", showProgress),
      go.importObject,
    );
    go.run(wasm.instance);
  } catch (err) {
    console.error("loading wasm failed: ", err);
  }
}

function appCanLoadWebAssembly() {
  if (
    /bot|googlebot|crawler|spider|robot|crawling/i.test(navigator.userAgent)
  ) {
    return false;
  }

  const urlParams = new URLSearchParams(window.location.search);
  return urlParams.get("wasm") !== "false";
}

async function fetchWithProgress(url, progress) {
  const response = await fetch(url);

  let contentLength;
  try {
    contentLength = response.headers.get(appWasmContentLengthHeader);
  } catch { }
  if (!appWasmContentLengthHeader || !contentLength) {
    contentLength = response.headers.get("Content-Length");
  }
  if (!contentLength) {
    contentLength = "60000000"; // 60 mb default
  }
  let contentEncoding = response.headers.get("Content-Encoding");

  let total = parseInt(contentLength, 10);
  if (contentEncoding) {
    total = total * 5; // we assume that compression reduces the size 5x
  }
  let loaded = 0;

  const progressHandler = function (loaded, total) {
    progress(Math.min(99, Math.max(1, Math.round((loaded * 100) / total)))); // must be 1-99
  };

  var res = new Response(
    new ReadableStream(
      {
        async start(controller) {
          var reader = response.body.getReader();
          for (; ;) {
            var { done, value } = await reader.read();

            if (done) {
              progressHandler(total, total);
              break;
            }

            loaded += value.byteLength;
            progressHandler(loaded, total);
            controller.enqueue(value);
          }
          controller.close();
        },
      },
      {
        status: response.status,
        statusText: response.statusText,
      }
    )
  );

  for (var pair of response.headers.entries()) {
    res.headers.set(pair[0], pair[1]);
  }

  return res;
}
