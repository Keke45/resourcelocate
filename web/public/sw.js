// Service worker: lets ResourceLocate be installed and keeps it quick on
// slow mobile data. The app shell (page, scripts, styles, fonts, icons) is
// cached; API calls always go to the network so data is never stale.
// Bump VERSION to drop old caches after a change to this file.
const VERSION = "v1";
const SHELL = `rl-shell-${VERSION}`;
const PRECACHE = ["/", "/manifest.webmanifest", "/icons/icon-192.png", "/icons/icon-512.png"];

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(SHELL).then((c) => c.addAll(PRECACHE)).then(() => self.skipWaiting()));
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== SHELL).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (event) => {
  const req = event.request;
  if (req.method !== "GET") return;
  const url = new URL(req.url);

  // Data and photos: network only.
  if (url.origin === location.origin && url.pathname.startsWith("/api/")) return;

  // Pages: try the network for the latest version, fall back to the cached
  // shell when offline. Share links (/l/...) are pages too.
  if (req.mode === "navigate") {
    event.respondWith(
      fetch(req)
        .then((res) => {
          if (res.ok && url.pathname === "/") caches.open(SHELL).then((c) => c.put("/", res.clone()));
          return res;
        })
        .catch(() => caches.match("/")),
    );
    return;
  }

  // Built assets have hashed names and fonts never change: cache first.
  const cacheFirst = (url.origin === location.origin && (url.pathname.startsWith("/assets/") || url.pathname.startsWith("/icons/")))
    || url.hostname === "fonts.googleapis.com" || url.hostname === "fonts.gstatic.com";
  if (cacheFirst) {
    event.respondWith(
      caches.match(req).then((hit) => hit || fetch(req).then((res) => {
        if (res.ok || res.type === "opaque") {
          const copy = res.clone();
          caches.open(SHELL).then((c) => c.put(req, copy));
        }
        return res;
      })),
    );
  }
});
