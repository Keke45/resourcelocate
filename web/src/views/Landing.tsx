import { useEffect, useState } from "react";
import { api } from "../api";
import { kg, useMeta } from "../meta";
import type { Stats } from "../types";

type Props = { onStart: () => void; onLogin: () => void };

const principles = [
  {
    title: "Use before waste",
    body: "Reuse it as it is if you can. Repair or remake it if you can't. Recycle it if neither works. Disposal is the last resort, never the default.",
  },
  {
    title: "Connection is the missing piece",
    body: "The material already exists and so does the need. What's usually missing is a way for the two to find each other, so that's what we build.",
  },
  {
    title: "Count only what really happened",
    body: "A listing on its own isn't impact. We count what was actually handed over, once the receiver confirms what they did with it.",
  },
  {
    title: "Safe by design",
    body: "Batteries, chemicals and other hazardous items are only offered to recyclers who can handle them, never to the general public.",
  },
  {
    title: "Local first",
    body: "Matches favour your own area. Short trips keep pickups cheap, quick and low in emissions.",
  },
];

const steps = [
  { title: "List what you don't need", body: "Say what it is, count the items or estimate the weight, and add a photo or two if you like." },
  { title: "Save what you're looking for", body: "Describe a need once. It stays on your dashboard, and new matching listings show up there as they're posted." },
  { title: "Hand it over and record it", body: "The giver accepts and confirms the pickup. The receiver says what became of it. That's what gets counted." },
];

const audiences = [
  { title: "Givers", body: "Households, businesses, workshops, schools and other institutions with materials they no longer need." },
  { title: "Seekers", body: "Individuals, students, small businesses and community organisations who need materials for projects, repairs and production." },
  { title: "Recyclers", body: "Collectors and processors who can turn what can't be reused into something useful, and handle what needs care." },
];

// Landing is the public front page: the problem, the philosophy behind the
// system, how it works, and live community totals.
export default function Landing({ onStart, onLogin }: Props) {
  const meta = useMeta();
  const [stats, setStats] = useState<Stats | null>(null);

  useEffect(() => {
    api.stats().then(setStats, () => setStats(null));
  }, []);

  // Only show totals once there is something to show; a new deployment
  // starts at zero and that is not worth a headline.
  const hasActivity = stats && (stats.openListings > 0 || stats.openNeeds > 0 || stats.totals.handedOn.handOvers > 0);

  return (
    <div className="landing">
      <section className="hero">
        <p className="eyebrow">Waste-to-resource exchange</p>
        <h1 className="hero-title">What one person throws away, someone nearby is looking for.</h1>
        <p className="hero-lede">
          ResourceLocate connects people who have unwanted materials with people who can use them, so useful things get a
          second life instead of ending up dumped, burned or in landfill.
        </p>
        <div className="actions">
          <button className="btn btn-primary btn-large" onClick={onStart}>Create a free account</button>
          <button className="btn btn-large" onClick={onLogin}>{meta.demoLogins ? "Log in or try the demo" : "Log in"}</button>
        </div>
        {hasActivity && (
          <dl className="hero-stats">
            <div><dt>Open listings</dt><dd className="num">{stats.openListings}</dd></div>
            <div><dt>People looking</dt><dd className="num">{stats.openNeeds}</dd></div>
            <div><dt>Finished hand-overs</dt><dd className="num">{stats.totals.handedOn.handOvers}</dd></div>
            {stats.totals.handedOn.kg > 0 && <div><dt>Material handed on</dt><dd className="num">{kg(stats.totals.handedOn.kg)}</dd></div>}
            {stats.totals.handedOn.pieces > 0 && <div><dt>Items handed on</dt><dd className="num">{stats.totals.handedOn.pieces}</dd></div>}
          </dl>
        )}
      </section>

      <section className="panel problem" aria-labelledby="problem-title">
        <h2 id="problem-title" className="landing-title">The problem</h2>
        <div className="problem-body">
          <p>
            Every day, households, businesses, workshops and institutions discard materials they no longer need. Some of
            these materials still have value. Others can be recycled or transformed into something useful.
          </p>
          <p>
            At the same time, individuals, students, small businesses and community organisations need materials for
            projects, repairs, production and other activities.
          </p>
          <p className="problem-key">
            The problem is that people who have unwanted materials often struggle to connect with people who could use them.
          </p>
        </div>
      </section>

      <section aria-labelledby="philosophy-title">
        <h2 id="philosophy-title" className="landing-title">What we believe</h2>
        <ul className="principles">
          {principles.map((p, i) => (
            <li key={p.title} className="panel principle">
              <span className="principle-n num">{String(i + 1).padStart(2, "0")}</span>
              <h3>{p.title}</h3>
              <p>{p.body}</p>
            </li>
          ))}
        </ul>
      </section>

      <section aria-labelledby="how-title">
        <h2 id="how-title" className="landing-title">How it works</h2>
        <ol className="how">
          {steps.map((s, i) => (
            <li key={s.title} className="panel how-step">
              <span className="how-n">{i + 1}</span>
              <h3>{s.title}</h3>
              <p>{s.body}</p>
            </li>
          ))}
        </ol>
      </section>

      <section aria-labelledby="who-title">
        <h2 id="who-title" className="landing-title">Who it's for</h2>
        <ul className="audiences">
          {audiences.map((a) => (
            <li key={a.title} className="panel">
              <h3>{a.title}</h3>
              <p>{a.body}</p>
            </li>
          ))}
        </ul>
        <p className="fine landing-note">One account does both: give away what you don't need and find what you do.</p>
      </section>

      <section className="panel method" aria-labelledby="method-title">
        <h2 id="method-title" className="landing-title">What we count</h2>
        <p>
          Simply what changed hands: the kilograms and pieces given away and claimed through finished hand-overs. A
          hand-over finishes when the giver confirms the pickup and the receiver says what they did with it. Listings on
          their own don't count.
        </p>
      </section>

      <section className="panel cta">
        <h2 className="landing-title">Have something to pass on, or something you need?</h2>
        <p>It takes a minute to sign up. Piloted in Lusaka, and built to work in any city.</p>
        <div className="actions">
          <button className="btn btn-primary btn-large" onClick={onStart}>Get started</button>
        </div>
      </section>
    </div>
  );
}
