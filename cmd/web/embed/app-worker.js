const cacheName = "app-" + "{{.Version}}";
// resourcesToCache are the files of the app, which are cached when the
// service worker is installed and then loaded from the cache.
const resourcesToCache = {{.ResourcesToCache}};
// externalResourcesToCache are the files from other origins (CDNs) that the
// pages load, such as fonts and the Vue.js of content sites. They are also
// cached on install if possible, but without failing the install if not.
// They are requested with CORS (which these CDNs support), since the opaque
// responses of no-cors requests can not be checked, and browsers count them
// as large in the storage quota.
const externalResourcesToCache = {{.ExternalResourcesToCache}};
// cacheOrigins are the other origins whose files are cached as they are
// loaded, such as font files, along with those of the app's origin, so
// that they also load offline. They are loaded from the network first, so
// that they are up to date online.
const cacheOrigins = {{.CacheOrigins}};

// preCached is the set of the URLs that are cached on install.
const preCached = new Set(
  resourcesToCache.concat(externalResourcesToCache).map((r) => new URL(r, self.location).href)
);

self.addEventListener("install", (event) => {
  console.log("installing app worker {{.Version}}");

  event.waitUntil(
    caches
      .open(cacheName)
      .then((cache) => {
        return cache.addAll(resourcesToCache).then(() =>
          Promise.all(
            externalResourcesToCache.map((url) =>
              cache.add(new Request(url, { mode: "cors" })).catch((err) => {
                console.warn("app worker could not cache " + url + ": ", err);
              })
            )
          )
        );
      })
      .then(() => {
        self.skipWaiting();
      })
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys().then((keyList) => {
      return Promise.all(
        keyList.map((key) => {
          if (key !== cacheName) {
            return caches.delete(key);
          }
        })
      );
    })
  );
  console.log("app worker {{.Version}} is activated");
});

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.method !== "GET" || request.headers.has("range")) {
    return;
  }
  // Pages are loaded from the network first, like the files that are not
  // cached on install (see fetchAndCache), so that they are up to date.
  if (preCached.has(request.url) && request.mode !== "navigate") {
    event.respondWith(
      caches.match(request).then((response) => {
        return response || fetchAndCache(request);
      })
    );
    return;
  }
  const origin = new URL(request.url).origin;
  if (origin === self.location.origin || cacheOrigins.includes(origin)) {
    event.respondWith(fetchAndCache(request));
  }
});

// fetchAndCache returns the response to the given request from the network,
// caching a copy of it, or the cached response if the network fails, so that
// the pages and files that have been loaded also load offline. Opaque and
// partial responses are not cached.
async function fetchAndCache(request) {
  try {
    const response = await fetch(request);
    if (response.status === 200 && (response.type === "basic" || response.type === "cors")) {
      const copy = response.clone();
      caches.open(cacheName).then((cache) => cache.put(request, copy)).catch(() => { });
    }
    return response;
  } catch (err) {
    // A page is the same with any query, such as ?wasm=false.
    const response = (await caches.match(request)) ||
      (request.mode === "navigate" && (await caches.match(request, { ignoreSearch: true })));
    if (response) {
      return response;
    }
    throw err;
  }
}

self.addEventListener("push", (event) => {
  if (!event.data || !event.data.text()) {
    return;
  }

  const notification = JSON.parse(event.data.text());
  if (!notification) {
    return;
  }

  const title = notification.title;
  delete notification.title;

  if (!notification.data) {
    notification.data = {};
  }
  let actions = [];
  for (let i in notification.actions) {
    const action = notification.actions[i];

    actions.push({
      action: action.action,
      path: action.path,
    });

    delete action.path;
  }
  notification.data.app = {
    path: notification.path,
    actions: actions,
  };
  delete notification.path;

  event.waitUntil(self.registration.showNotification(title, notification));
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();

  const notification = event.notification;
  let path = notification.data.app.path;

  for (let i in notification.data.app.actions) {
    const action = notification.data.app.actions[i];
    if (action.action === event.action) {
      path = action.path;
      break;
    }
  }

  event.waitUntil(
    clients
      .matchAll({
        type: "window",
      })
      .then((clientList) => {
        for (var i = 0; i < clientList.length; i++) {
          let client = clientList[i];
          if ("focus" in client) {
            client.focus();
            client.postMessage({
              app: {
                type: "notification",
                path: path,
              },
            });
            return;
          }
        }

        if (clients.openWindow) {
          return clients.openWindow(path);
        }
      })
  );
});
