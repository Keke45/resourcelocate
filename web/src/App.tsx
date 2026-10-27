import { useCallback, useEffect, useState } from "react";
import { api, setSignedOutHandler } from "./api";
import { MetaContext, UserContext, roleLabels } from "./meta";
import type { Meta, User } from "./types";
import Landing from "./views/Landing";
import AuthView from "./views/AuthView";
import DashboardView from "./views/DashboardView";
import FindView from "./views/FindView";
import ListView from "./views/ListView";
import ActivityView from "./views/ActivityView";
import SettingsView from "./views/SettingsView";
import SharedListingView from "./views/SharedListingView";
import { InstallButton } from "./components/install";
import { myTurn } from "./views/DashboardView";

type PublicPage = "home" | "login" | "signup";
type Tab = "dashboard" | "find" | "give" | "handovers" | "settings";
type Page = PublicPage | Tab;

// Simple line icons for the navigation, drawn on a 24px grid.
const icons: Record<Tab, string> = {
  dashboard: "M4 4h7v7H4zM13 4h7v4h-7zM13 10h7v10h-7zM4 13h7v7H4z",
  find: "M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14zM16 16l4.5 4.5",
  give: "M12 5v14M5 12h14",
  handovers: "M4 8h13l-3-3M20 16H7l3 3",
  settings: "M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6zM12 3v2M12 19v2M3 12h2M19 12h2M5.6 5.6l1.4 1.4M17 17l1.4 1.4M5.6 18.4 7 17M17 7l1.4-1.4",
};

const tabs: { id: Tab; label: string; hint: string }[] = [
  { id: "dashboard", label: "Dashboard", hint: "Your activity" },
  { id: "find", label: "Find", hint: "Materials near you" },
  { id: "give", label: "Give", hint: "List unwanted material" },
  { id: "handovers", label: "Hand-overs", hint: "Requests and pickups" },
  { id: "settings", label: "Settings", hint: "Profile and password" },
];

function NavIcon({ tab }: { tab: Tab }) {
  return (
    <svg className="tab-icon" viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor"
      strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d={icons[tab]} />
    </svg>
  );
}

// initials turns "Mwila Banda" into "MB" for the account badge.
const initials = (name: string) =>
  name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0].toUpperCase()).join("");
const pages: Page[] = ["home", "login", "signup", ...tabs.map((t) => t.id)];

// The page lives in the URL hash so a refresh or shared link keeps it.
function pageFromHash(): Page {
  const h = window.location.hash.slice(1);
  return pages.includes(h as Page) ? (h as Page) : "home";
}

// A shared listing link looks like /l/{id}; it opens that listing for
// anyone, signed in or not.
function sharedFromPath(): string | null {
  const m = window.location.pathname.match(/^\/l\/([^/]+)$/);
  return m ? decodeURIComponent(m[1]) : null;
}

// afterLogin remembers which shared listing to return to once a visitor has
// logged in or signed up. Storage can be unavailable (private modes), in
// which case they simply land on the dashboard.
const afterLogin = {
  set(id: string) { try { sessionStorage.setItem("rl-after-login", id); } catch { /* not available */ } },
  take(): string | null {
    try {
      const id = sessionStorage.getItem("rl-after-login");
      sessionStorage.removeItem("rl-after-login");
      return id;
    } catch {
      return null;
    }
  },
};

const isTab = (p: Page): p is Tab => tabs.some((t) => t.id === p);

export default function App() {
  const [meta, setMeta] = useState<Meta | null>(null);
  // undefined while we ask the server who is signed in; null when nobody is.
  const [user, setUser] = useState<User | null | undefined>(undefined);
  const [loadError, setLoadError] = useState("");
  const [page, setPage] = useState<Page>(pageFromHash);
  const [shared, setShared] = useState<string | null>(sharedFromPath);
  const [toast, setToast] = useState("");
  // Bumped after any write so every view refetches what it shows.
  const [version, setVersion] = useState(0);

  useEffect(() => {
    api.meta().then(setMeta, (e: Error) => setLoadError(e.message));
    api.me().then(setUser, () => setUser(null));
    setSignedOutHandler(() => setUser(null));
  }, []);

  // Keep the page in step with the back and forward buttons.
  useEffect(() => {
    const onNav = () => {
      setPage(pageFromHash());
      setShared(sharedFromPath());
    };
    window.addEventListener("hashchange", onNav);
    window.addEventListener("popstate", onNav);
    return () => {
      window.removeEventListener("hashchange", onNav);
      window.removeEventListener("popstate", onNav);
    };
  }, []);

  // How many hand-overs are waiting on this user, shown as a badge in the
  // navigation. Refetched after every write, like the views.
  const [waiting, setWaiting] = useState(0);
  useEffect(() => {
    if (!user) return setWaiting(0);
    api.transfers().then((ts) => setWaiting(ts.filter(myTurn).length), () => {});
  }, [user, version]);

  useEffect(() => {
    if (!toast) return;
    const t = setTimeout(() => setToast(""), 3200);
    return () => clearTimeout(t);
  }, [toast]);

  const go = useCallback((p: Page) => {
    setPage(p);
    setShared(null);
    history.pushState(null, "", "/#" + p);
    window.scrollTo({ top: 0 });
  }, []);

  const openShared = useCallback((id: string) => {
    setShared(id);
    history.pushState(null, "", "/l/" + encodeURIComponent(id));
    window.scrollTo({ top: 0 });
  }, []);

  // changed is called by a view after a successful write: refetch everything,
  // confirm with a toast, and optionally move to the page that shows the result.
  const changed = useCallback((message: string, next?: Tab) => {
    setVersion((v) => v + 1);
    setToast(message);
    if (next) go(next);
  }, [go]);

  const signedIn = (u: User) => {
    setUser(u);
    const back = afterLogin.take();
    if (back) openShared(back);
    else go("dashboard");
  };

  async function logout() {
    await api.logout().catch(() => {});
    setToast("");
    setUser(null);
    go("home");
  }

  if (loadError) return <p className="banner banner-error boot">{loadError}</p>;
  if (!meta || user === undefined) return <p className="loading boot">Loading…</p>;

  // Signed-out visitors only see the public pages; signed-in users skip them.
  const shown: Page = user ? (isTab(page) ? page : "dashboard") : isTab(page) ? "login" : page;

  const brand = (
    <a className="brand" href="#home" onClick={(e) => { e.preventDefault(); go(user ? "dashboard" : "home"); }}>
      <span className="brand-mark" aria-hidden="true" />
      <span className="brand-name">Resource<span>Locate</span></span>
    </a>
  );

  const main = (
    <main className={`content ${shown === "home" && !shared ? "content-wide" : ""}`}>
      {shared && (
        <SharedListingView id={shared} user={user}
          onLogin={() => { afterLogin.set(shared); go("login"); }}
          onSignup={() => { afterLogin.set(shared); go("signup"); }}
          onRequested={(m) => changed(m, "handovers")}
          onBrowse={() => go(user ? "find" : "home")} />
      )}
      {!shared && !user && shown === "home" && <Landing onStart={() => go("signup")} onLogin={() => go("login")} />}
      {!shared && !user && (shown === "login" || shown === "signup") && (
        <AuthView mode={shown} onSwitch={(m) => go(m)} onSignedIn={signedIn} />
      )}
      {!shared && user && (
        <UserContext.Provider value={user}>
          {shown === "dashboard" && <DashboardView version={version} onChanged={changed} go={go} />}
          {shown === "find" && <FindView version={version} onChanged={changed} />}
          {shown === "give" && <ListView onChanged={changed} />}
          {shown === "handovers" && <ActivityView version={version} onChanged={changed} />}
          {shown === "settings" && <SettingsView onUserChanged={setUser} onChanged={(m) => setToast(m)} />}
        </UserContext.Provider>
      )}
    </main>
  );

  const footer = (
    <footer className="footer">
      ResourceLocate · Built for any city where useful materials go to waste
    </footer>
  );

  const toastEl = <div className="toast" role="status" aria-live="polite" hidden={!toast}>{toast}</div>;

  // Signed in: a sidebar on wider screens. On phones the same sidebar folds
  // into a top bar, and its navigation becomes a fixed bottom tab bar.
  if (user) {
    return (
      <MetaContext.Provider value={meta}>
        <div className="app">
          <aside className="sidebar">
            {brand}
            <nav className="tabs" aria-label="Sections">
              {tabs.map((t) => (
                <button key={t.id} className={`tab tab-${t.id}`} aria-current={!shared && shown === t.id ? "page" : undefined} onClick={() => go(t.id)}>
                  <NavIcon tab={t.id} />
                  <span className="tab-text">
                    <span className="tab-label">{t.label}</span>
                    <span className="tab-hint">{t.hint}</span>
                  </span>
                  {t.id === "handovers" && waiting > 0 && (
                    <span className="tab-badge num" aria-label={`${waiting} waiting on you`}>{waiting}</span>
                  )}
                </button>
              ))}
            </nav>
            <InstallButton className="btn btn-quiet btn-small side-install" />
            <div className="side-account">
              {/* On phones Settings leaves the bottom bar, so the initials open it. */}
              <button className="avatar" onClick={() => go("settings")} aria-label="Settings" title="Settings">{initials(user.name)}</button>
              <span className="side-who">
                <span className="account-name" title={user.email}>{user.name}</span>
                <span className="fine">{roleLabels[user.role]} · {user.area}</span>
              </span>
              <button className="btn btn-quiet btn-small" onClick={logout}>Log out</button>
            </div>
          </aside>
          <div className="app-main">
            {main}
            {footer}
          </div>
          {toastEl}
        </div>
      </MetaContext.Provider>
    );
  }

  return (
    <MetaContext.Provider value={meta}>
      <div className="shell">
        <header className="masthead">
          {brand}
          <div className="account">
            <InstallButton className="btn btn-quiet public-install" />
            <button className="btn btn-quiet" onClick={() => go("login")}>Log in</button>
            <button className="btn btn-primary" onClick={() => go("signup")}>Sign up</button>
          </div>
        </header>
        {main}
        {footer}
        {toastEl}
      </div>
    </MetaContext.Provider>
  );
}
