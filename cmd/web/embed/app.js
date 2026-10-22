// -----------------------------------------------------------------------------
// app
// -----------------------------------------------------------------------------
var appNav = function () { };
var appOnUpdate = function () { };
var appOnAppInstallChange = function () { };

const appEnv = {{.Env }};
const appWasmContentLengthHeader = "{{.WasmContentLengthHeader}}";

// appSharedDownloadMaxSize is the largest transfer size of the bundle for
// which appShowDownloadProgress reads it, since the browser then shares one
// download with the module load (see there).
const appSharedDownloadMaxSize = 8000000;

window.wasm = null;

let appServiceWorkerRegistration;
let deferredPrompt = null;

appLoadFonts();
appWhenLoaded(appInitServiceWorker);
appWatchForUpdate();
appWatchForInstallable();
appInitWebAssembly();

// -----------------------------------------------------------------------------
// Fonts
// -----------------------------------------------------------------------------

// appLoadFonts starts loading the regular and italic faces of the fonts of
// the app (with the Latin runes; the faces have all weights), so that they
// load while the app loads, without waiting for them. The app measures its
// texts with them, and texts measured before their face has loaded are
// measured with a fallback font, so the app lays out all of its texts again
// when the face loads, which made the first page with code or italic text
// take twice as long. Regular Noto Sans also loads the faces of the
// characters of the FontText web setting (the cogentcore-font-text meta
// element), such as "āΩ" for Latin Extended and Greek.
function appLoadFonts() {
  if (!document.fonts) {
    return;
  }
  const text = document.querySelector('meta[name="cogentcore-font-text"]')?.content || '';
  document.fonts.load('16px "Noto Sans"', 'a' + text).catch(() => { });
  for (const font of ['italic 16px "Noto Sans"', '16px "Roboto Mono"', 'italic 16px "Roboto Mono"']) {
    document.fonts.load(font).catch(() => { });
  }
}

// -----------------------------------------------------------------------------
// Service Worker
// -----------------------------------------------------------------------------

// appWhenLoaded calls f once the app has rendered, which removes the loader,
// or now if there is no loader. The service worker is registered then, since
// it downloads the files of the app to cache them, which would otherwise slow
// down the first load of the app.
function appWhenLoaded(f) {
  const loader = document.getElementById("app-wasm-loader");
  if (!loader || !loader.parentNode) {
    f();
    return;
  }
  const observer = new MutationObserver(() => {
    if (!loader.isConnected) {
      observer.disconnect();
      f();
    }
  });
  observer.observe(loader.parentNode, { childList: true });
}

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
// Web Assembly
// -----------------------------------------------------------------------------

async function appInitWebAssembly() {
  const loader = document.getElementById("app-wasm-loader");

  if (!appCanLoadWebAssembly()) {
    if (loader.classList.contains("cc-site")) {
      // A content website works without the app, so we keep it.
      loader.querySelector(".cc-loading").remove();
    } else {
      loader.remove();
    }
    return;
  }

  {{- if .JS}}
  await appInitGoScript(loader);
  return;
  {{- end}}

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

// appInitGoScript loads the app compiled to JavaScript with GoScript
// (the js platform), showing download progress in the loader.
async function appInitGoScript(loader) {
  const loaderLabel = document.getElementById("app-wasm-loader-label");
  const loaderProgress = document.getElementById("app-wasm-loader-progress");
  const showProgress = (progress) => {
    if (loaderLabel) {
      loaderLabel.innerText = progress + "%";
    }
    if (loaderProgress) {
      loaderProgress.value = progress / 100;
    }
  };
  // Without a known progress, the progress bar shows that the app is loading.
  const showLoading = () => {
    if (loaderLabel) {
      loaderLabel.innerText = "";
    }
    if (loaderProgress) {
      loaderProgress.removeAttribute("value");
    }
  };
  try {
    showProgress(1);
    const url = new URL(document.documentElement.dataset.basePath + "app.gs.js", document.baseURI).href;
    // Without the service worker, the bundle comes from the network, so we
    // also read it with fetch to show progress.
    if (!navigator.serviceWorker || !navigator.serviceWorker.controller) {
      appShowDownloadProgress(url, showProgress).then((shown) => shown || showLoading());
    } else {
      showLoading();
    }
    // We import the bundle URL itself, which the page already started to
    // load with modulepreload, so that the browser compiles it while it
    // downloads and reuses the code it compiled on later loads.
    await import(url);
  } catch (err) {
    console.error("loading app failed: ", err);
  }
}

// appShowDownloadProgress reads the response of the given URL, calling
// progress with the percentage downloaded, without keeping the data.
// It returns whether it read the response.
async function appShowDownloadProgress(url, progress) {
  try {
    // The browser shares one download between this request and the module
    // load through its HTTP cache only if the response can be cached, so we
    // check that first, to never download the bundle twice. (The query makes
    // the browser send the HEAD request without waiting for the download.)
    // The browser also does not cache a response larger than a fraction of
    // its cache, such as 10 MB of 80 MB, so a larger bundle is not shared.
    const head = await fetch(url + "?head", { method: "HEAD", cache: "no-store" });
    if (!/max-age=[1-9]|immutable/.test(head.headers.get("Cache-Control") || "")) {
      return false;
    }
    const size = parseInt(head.headers.get("Content-Length") || "0", 10);
    if (!(size > 0 && size <= appSharedDownloadMaxSize)) {
      return false;
    }
    const response = await fetch(url);
    const total = appResponseSize(response);
    const reader = response.body.getReader();
    let loaded = 0;
    for (; ;) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      loaded += value.byteLength;
      progress(Math.min(99, Math.max(1, Math.round((loaded * 100) / total)))); // must be 1-99
    }
    return true;
  } catch (err) {
    console.warn("showing download progress failed: ", err);
    return false;
  }
}

// appResponseSize returns the estimated size in bytes of the decoded body of
// the given response.
function appResponseSize(response) {
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
  let total = parseInt(contentLength, 10);
  if (response.headers.get("Content-Encoding")) {
    total = total * 5; // we assume that compression reduces the size 5x
  }
  return total;
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

  let total = appResponseSize(response);
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
