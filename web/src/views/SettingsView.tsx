import { useState } from "react";
import { api } from "../api";
import { roleLabels, useMeta, useUser } from "../meta";
import type { User } from "../types";

type Props = { onUserChanged: (u: User) => void; onChanged: (message: string) => void };

// SettingsView lets the signed-in user change their name, area and
// password. Email and role stay as they were at sign-up.
export default function SettingsView({ onUserChanged, onChanged }: Props) {
  const meta = useMeta();
  const user = useUser();
  return (
    <div className="settings">
      <header className="view-head">
        <h1 className="view-title">Settings</h1>
        <p className="view-lede">Signed in as {user.email} ({roleLabels[user.role].toLowerCase()}).</p>
      </header>
      <ProfileForm areas={meta.areas} onSaved={(u) => { onUserChanged(u); onChanged("Profile saved."); }} />
      <PasswordForm onSaved={() => onChanged("Password changed. Other devices have been logged out.")} />
    </div>
  );
}

function ProfileForm({ areas, onSaved }: { areas: string[]; onSaved: (u: User) => void }) {
  const user = useUser();
  const [name, setName] = useState(user.name);
  const [area, setArea] = useState(user.area);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const unchanged = name.trim() === user.name && area === user.area;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      onSaved(await api.updateProfile({ name, area }));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="panel form settings-card" onSubmit={submit} aria-labelledby="profile-title">
      <h2 id="profile-title" className="section-title">Profile</h2>
      <div className="field">
        <label htmlFor="s-name">Your name or organisation</label>
        <input id="s-name" value={name} onChange={(e) => setName(e.target.value)} required maxLength={80} autoComplete="name" />
        <span className="fine">Shown on new listings and requests.</span>
      </div>
      <div className="field">
        <label htmlFor="s-area">Your area</label>
        <select id="s-area" value={area} onChange={(e) => setArea(e.target.value)}>
          {areas.map((a) => <option key={a}>{a}</option>)}
        </select>
        <span className="fine">Used as the starting point when you search or list.</span>
      </div>
      {error && <p className="banner banner-error">{error}</p>}
      <div className="actions">
        <button className="btn btn-primary" disabled={busy || unchanged}>{busy ? "Saving…" : "Save profile"}</button>
      </div>
    </form>
  );
}

function PasswordForm({ onSaved }: { onSaved: () => void }) {
  const user = useUser();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const mismatch = confirm !== "" && confirm !== next;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (next !== confirm) return;
    setBusy(true);
    setError("");
    try {
      await api.changePassword(current, next);
      setCurrent("");
      setNext("");
      setConfirm("");
      onSaved();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="panel form settings-card" onSubmit={submit} aria-labelledby="password-title">
      <h2 id="password-title" className="section-title">Change password</h2>
      {/* Lets password managers tie the new password to this account. */}
      <input type="email" autoComplete="username" value={user.email} readOnly hidden />
      <div className="field">
        <label htmlFor="s-current">Current password</label>
        <input id="s-current" type="password" value={current} onChange={(e) => setCurrent(e.target.value)} required autoComplete="current-password" />
      </div>
      <div className="field">
          <label htmlFor="s-new">New password</label>
          <input id="s-new" type="password" value={next} onChange={(e) => setNext(e.target.value)} required minLength={8} autoComplete="new-password" />
          <span className="fine">At least 8 characters.</span>
      </div>
      <div className="field">
          <label htmlFor="s-confirm">Type it again</label>
          <input id="s-confirm" type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} required autoComplete="new-password"
            aria-invalid={mismatch} />
          {mismatch && <span className="fine field-error">The two don't match.</span>}
      </div>
      <p className="fine">Changing it logs you out on every other phone or computer.</p>
      {error && <p className="banner banner-error">{error}</p>}
      <div className="actions">
        <button className="btn btn-primary" disabled={busy || mismatch}>{busy ? "Saving…" : "Change password"}</button>
      </div>
    </form>
  );
}
