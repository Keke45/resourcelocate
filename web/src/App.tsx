import { useCallback, useEffect, useState } from "react";
import { api, setSignedOutHandler } from "./api";
import { MetaContext, UserContext } from "./meta";
import type { Meta, User } from "./types";
import Landing from "./views/Landing";
import AuthView from "./views/AuthView";
import DashboardView from "./views/DashboardView";
import FindView from "./views/FindView";
import ListView from "./views/ListView";
import ActivityView from "./views/ActivityView";

type PublicPage = "home" | "login" | "signup";
type Tab = "dashboard" | "find" | "give" | "handovers";
type Page = PublicPage | Tab;

const tabs: { id: Tab; label: string; hint: string }[] = [
  { id: "dashboard", label: "Dashboard", hint: "Your activity" },
  { id: "find", label: "Find", hint: "Materials near you" },
  { id: "give", label: "Give", hint: "List unwanted material" },
  { id: "handovers", label: "Hand-overs", hint: "Requests and pickups" },
];
const pages: Page[] = ["home", "login", "signup", ...tabs.map((t) => t.id)];

// The page lives in the URL hash so a refresh or shared link keeps it.
function pageFromHash(): Page {
  const h = window.location.hash.slice(1);
  return pages.includes(h as Page) ? (h as Page) : "home";
}

const isTab = (p: Page): p is Tab => tabs.some((t) => t.id === p);

export default function App() {
  const [meta, setMeta] = useState<Meta | null>(null);
  // undefined while we ask the server who is signed in; null when nobody is.
  const [user, setUser] = useState<User | null | undefined>(undefined);
  const [loadError, setLoadError] = useState("");
  const [page, setPage] = useState<Page>(pageFromHash);
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
    const onHash = () => setPage(pageFromHash());
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  useEffect(() => {
    if (!toast) return;
    const t = setTimeout(() => setToast(""), 3200);
    return () => clearTimeout(t);
  }, [toast]);

  const go = useCallback((p: Page) => {
    setPage(p);
    history.pushState(null, "", "#" + p);
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
    go("dashboard");
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

  return (
    <MetaContext.Provider value={meta}>
      <div className="shell">
        {user ? (
          <header className="masthead">
            {brand}
            <nav className="tabs" aria-label="Sections">
              {tabs.map((t) => (
                <button key={t.id} className="tab" aria-current={shown === t.id ? "page" : undefined} onClick={() => go(t.id)}>
                  <span className="tab-label">{t.label}</span>
                  <span className="tab-hint">{t.hint}</span>
                </button>
              ))}
            </nav>
            <div className="account">
              <span className="account-name" title={user.email}>{user.name}</span>
              <button className="btn btn-quiet" onClick={logout}>Log out</button>
            </div>
          </header>
        ) : (
          <header className="masthead">
            {brand}
            <div className="account">
              <button className="btn btn-quiet" onClick={() => go("login")}>Log in</button>
              <button className="btn btn-primary" onClick={() => go("signup")}>Sign up</button>
            </div>
          </header>
        )}

        <main className={`content ${shown === "home" ? "content-wide" : ""}`}>
          {!user && shown === "home" && <Landing onStart={() => go("signup")} onLogin={() => go("login")} />}
          {!user && (shown === "login" || shown === "signup") && (
            <AuthView mode={shown} onSwitch={(m) => go(m)} onSignedIn={signedIn} />
          )}
          {user && (
            <UserContext.Provider value={user}>
              {shown === "dashboard" && <DashboardView version={version} onChanged={changed} go={go} />}
              {shown === "find" && <FindView version={version} onChanged={changed} />}
              {shown === "give" && <ListView onChanged={changed} />}
              {shown === "handovers" && <ActivityView version={version} onChanged={changed} />}
            </UserContext.Provider>
          )}
        </main>

        <footer className="footer">
          ResourceLocate · piloted in Lusaka, built for any city where useful materials go to waste.
        </footer>

        <div className="toast" role="status" aria-live="polite" hidden={!toast}>{toast}</div>
      </div>
    </MetaContext.Provider>
  );
}
