import { useState } from "react";
import { api } from "../api";
import { roleLabels, useMeta } from "../meta";
import type { Role, User } from "../types";

type Mode = "login" | "signup";
type Props = { mode: Mode; onSwitch: (m: Mode) => void; onSignedIn: (u: User) => void };

const roleHints: Record<Role, string> = {
  individual: "Students, makers and anyone acting for themselves.",
  business: "Shops, workshops, schools, groups and other organisations.",
  recycler: "Collectors and processors. Only recyclers can take items that need specialised handling.",
};

// AuthView is the login and sign-up page. Both create a session cookie on
// the server; the page never stores the password or a token itself.
export default function AuthView({ mode, onSwitch, onSignedIn }: Props) {
  const meta = useMeta();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<Role>("individual");
  const [area, setArea] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const signup = mode === "signup";

  // demo signs in as one of the shared demo accounts (when the server has
  // them on), so judges can try both sides of a hand-over without signing up.
  async function demo(as: "giver" | "seeker") {
    setBusy(true);
    setError("");
    try {
      onSignedIn(await api.demoLogin(as));
    } catch (err) {
      setError((err as Error).message);
      setBusy(false);
    }
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const u = signup ? await api.signup({ name, email, password, role, area }) : await api.login(email, password);
      onSignedIn(u);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="auth">
      <section className="panel auth-card" aria-labelledby="auth-title">
        <h1 id="auth-title" className="view-title">{signup ? "Create your account" : "Welcome back"}</h1>
        <p className="view-lede">
          {signup
            ? "One account lets you give away materials and find the ones you need."
            : "Log in to see your listings, needs and hand-overs."}
        </p>

        <form className="form" onSubmit={submit}>
          {signup && (
            <div className="field">
              <label htmlFor="a-name">Your name or organisation</label>
              <input id="a-name" value={name} onChange={(e) => setName(e.target.value)} required maxLength={80} autoComplete="name" />
            </div>
          )}
          <div className="field">
            <label htmlFor="a-email">Email</label>
            <input id="a-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoComplete="email" />
          </div>
          <div className="field">
            <label htmlFor="a-password">Password</label>
            <input id="a-password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required
              minLength={signup ? 8 : undefined} autoComplete={signup ? "new-password" : "current-password"} />
            {signup && <span className="fine">At least 8 characters.</span>}
          </div>
          {signup && (
            <>
              <fieldset className="field">
                <legend>You are</legend>
                <div className="role-options">
                  {meta.roles.map((r) => (
                    <label key={r} className={`role-option ${role === r ? "is-on" : ""}`}>
                      <input type="radio" name="role" value={r} checked={role === r} onChange={() => setRole(r)} />
                      <span>
                        <strong>{roleLabels[r]}</strong>
                        <span className="fine">{roleHints[r]}</span>
                      </span>
                    </label>
                  ))}
                </div>
              </fieldset>
              <div className="field">
                <label htmlFor="a-area">Your area</label>
                <select id="a-area" value={area} onChange={(e) => setArea(e.target.value)} required>
                  <option value="" disabled>Choose…</option>
                  {meta.areas.map((a) => <option key={a}>{a}</option>)}
                </select>
              </div>
            </>
          )}
          {error && <p className="banner banner-error">{error}</p>}
          <button className="btn btn-primary" disabled={busy}>
            {busy ? "One moment…" : signup ? "Create account" : "Log in"}
          </button>
        </form>

        {meta.demoLogins && (
          <div className="demo-box">
            <p><strong>Judging this project?</strong> Try it without signing up. Use one account to list something and the
              other to request it, in two browser windows or one after the other.</p>
            <div className="actions">
              <button className="btn" disabled={busy} onClick={() => demo("giver")}>Demo giver</button>
              <button className="btn" disabled={busy} onClick={() => demo("seeker")}>Demo seeker</button>
            </div>
            <p className="fine">The demo accounts are shared, so you may see what other visitors did.</p>
          </div>
        )}

        <p className="auth-switch">
          {signup ? "Already have an account? " : "New here? "}
          <button className="link" onClick={() => onSwitch(signup ? "login" : "signup")}>
            {signup ? "Log in" : "Create an account"}
          </button>
        </p>
      </section>
    </div>
  );
}
