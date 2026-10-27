import { useEffect, useState } from "react";

// Chrome, Edge and Samsung Internet fire beforeinstallprompt when the app
// can be installed; we keep the event so a button can show the prompt
// later. Safari has no such event: there, people use Share > Add to Home
// Screen, so no button appears. The listener is added at load, because the
// event can fire before React renders.
type InstallEvent = Event & { prompt: () => Promise<void>; userChoice: Promise<{ outcome: string }> };

let deferred: InstallEvent | null = null;
const listeners = new Set<() => void>();
const notify = () => listeners.forEach((f) => f());

window.addEventListener("beforeinstallprompt", (e) => {
  e.preventDefault();
  deferred = e as InstallEvent;
  notify();
});
window.addEventListener("appinstalled", () => {
  deferred = null;
  notify();
});

// InstallButton offers to install the app when the browser allows it.
export function InstallButton({ className = "btn btn-quiet btn-small" }: { className?: string }) {
  const [, rerender] = useState(0);
  useEffect(() => {
    const f = () => rerender((n) => n + 1);
    listeners.add(f);
    return () => { listeners.delete(f); };
  }, []);
  if (!deferred) return null;

  async function install() {
    const e = deferred;
    if (!e) return;
    await e.prompt();
    await e.userChoice;
    deferred = null;
    notify();
  }

  return (
    <button type="button" className={className} onClick={install} aria-label="Install app">
      <svg className="install-icon" viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" strokeWidth="1.8"
        strokeLinecap="round" strokeLinejoin="round"><path d="M12 4v11M7 10l5 5 5-5M5 20h14" /></svg>
      <span className="install-label">Install app</span>
    </button>
  );
}
