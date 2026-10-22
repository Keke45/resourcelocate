import { useEffect, useState } from "react";
import { api } from "../api";
import { ago, amount as describeAmount, roleLabels, useLabels, useMeta, useUser } from "../meta";
import type { Listing, Match } from "../types";
import { RouteChip } from "../components/bits";
import RequestSheet from "../components/RequestSheet";

type Props = { version: number; onChanged: (message: string, next?: "handovers" | "dashboard") => void };

// FindView is the seeker side: describe a need and see open listings ranked
// by the server. The search can be saved as a need, which stays on the
// dashboard and picks up new listings. Choosing a listing opens the request
// sheet.
export default function FindView({ version, onChanged }: Props) {
  const meta = useMeta();
  const labels = useLabels();
  const [material, setMaterial] = useState("");
  const [amount, setAmount] = useState("");
  // Seekers can ask by weight ("20 kg of offcuts") or by count ("10 monitors").
  const [unit, setUnit] = useState<"kg" | "pieces">("kg");
  const user = useUser();
  // Start from the user's own area; they can widen it to all of Lusaka.
  const [area, setArea] = useState(user.area);
  const [note, setNote] = useState("");
  const [saving, setSaving] = useState(false);
  const [matches, setMatches] = useState<Match[] | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState<Listing | null>(null);

  // Re-rank whenever the need changes, debounced so typing an amount does not
  // send a request per keystroke. `live` drops answers to outdated queries.
  useEffect(() => {
    let live = true;
    const t = setTimeout(() => {
      const q = unit === "kg" ? { kg: amount, pieces: "" } : { kg: "", pieces: amount };
      api.matches({ material, ...q, area }).then(
        (m) => live && (setMatches(m), setError("")),
        (e: Error) => live && setError(e.message),
      );
    }, 200);
    return () => {
      live = false;
      clearTimeout(t);
    };
  }, [material, amount, unit, area, version]);

  // With no need entered the list is just "what is open", so scores and
  // reasons are hidden.
  const hasNeed = Boolean(material || amount || area);

  async function saveNeed() {
    setSaving(true);
    setError("");
    try {
      const n = Number(amount) || 0;
      await api.saveNeed({ material, kg: unit === "kg" ? n : 0, pieces: unit === "pieces" ? Math.round(n) : 0, area, note });
      setNote("");
      onChanged("Need saved. New matches will show on your dashboard.");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="find">
      <section className="panel need" aria-labelledby="need-title">
        <h1 id="need-title" className="view-title">What do you need?</h1>
        <p className="view-lede">Tell us what you're looking for and we'll rank what's available across Lusaka.</p>
        <div className="need-grid">
          <div className="field">
            <label htmlFor="need-material">Material</label>
            <select id="need-material" value={material} onChange={(e) => setMaterial(e.target.value)}>
              <option value="">Any material</option>
              {meta.materials.map((m) => <option key={m.id} value={m.id}>{m.label}</option>)}
            </select>
          </div>
          <div className="field">
            <label htmlFor="need-kg">Amount needed</label>
            <div className="with-unit">
              <input id="need-kg" inputMode={unit === "kg" ? "decimal" : "numeric"} value={amount} placeholder="Any"
                onChange={(e) => setAmount(e.target.value.replace(unit === "kg" ? /[^0-9.]/g : /[^0-9]/g, ""))} />
              <select aria-label="Unit" value={unit} onChange={(e) => {
                setUnit(e.target.value as "kg" | "pieces");
                setAmount((a) => a.replace(/\..*$/, ""));
              }}>
                <option value="kg">kg</option>
                <option value="pieces">pieces</option>
              </select>
            </div>
          </div>
          <div className="field">
            <label htmlFor="need-area">Your area</label>
            <select id="need-area" value={area} onChange={(e) => setArea(e.target.value)}>
              <option value="">Anywhere in Lusaka</option>
              {meta.areas.map((a) => <option key={a}>{a}</option>)}
            </select>
          </div>
        </div>
        <div className="field">
          <label htmlFor="need-note">Note for your saved need (optional)</label>
          <input id="need-note" value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} placeholder="e.g. for a school science fair" />
        </div>
        <button className="btn" onClick={saveNeed} disabled={!material || saving}
          title={material ? undefined : "Choose a material to save this search"}>
          {saving ? "Saving…" : "Save as a need"}
        </button>
        {!material && <p className="fine">Choose a material to save this search to your dashboard.</p>}
        <p className="fine">
          Searching as {roleLabels[user.role].toLowerCase()}. Ranking: material 40, quantity 25, route fit 15, same area 20.
          Specialised-handling items only show for recyclers. Your own listings are left out.
        </p>
      </section>

      <section className="results" aria-live="polite" aria-labelledby="results-title">
        <h2 id="results-title" className="section-title">
          {matches === null ? "Loading…" : `${matches.length} ${hasNeed ? (matches.length === 1 ? "match" : "matches") : matches.length === 1 ? "open listing" : "open listings"}`}
        </h2>
        {error && <p className="banner banner-error">{error}</p>}
        {matches?.length === 0 && (
          <p className="empty">
            Nothing matches yet. Try any material or anywhere in Lusaka, or save this as a need and we'll show matches on your dashboard as they're listed.
          </p>
        )}
        <ul className="cards">
          {matches?.map((m, i) => (
            <li key={m.listing.id}>
              <button className="card listing" onClick={() => setOpen(m.listing)}>
                <div className="listing-top">
                  <span className="eyebrow">{labels.material(m.listing.material)}</span>
                  {hasNeed && (
                    <span className={`fit ${i === 0 ? "fit-best" : ""}`} title="Match score out of 100">
                      <span className="fit-bar"><span style={{ width: `${m.score}%` }} /></span>
                      <span className="num">{m.score}</span>
                    </span>
                  )}
                </div>
                <h3 className="listing-item">{m.listing.item}</h3>
                <p className="listing-meta">
                  <span className="num">{describeAmount(m.listing)}</span> · {m.listing.area} · {m.listing.providerName}
                </p>
                <div className="listing-foot">
                  <RouteChip route={m.listing.route} />
                  {hasNeed && m.reasons?.map((r) => <span key={r} className="reason">{r}</span>)}
                  <span className="when">{ago(m.listing.createdAt)}</span>
                </div>
              </button>
            </li>
          ))}
        </ul>
      </section>

      {open && (
        <RequestSheet
          listing={open}
          onClose={() => setOpen(null)}
          onRequested={(msg) => {
            setOpen(null);
            onChanged(msg, "handovers");
          }}
        />
      )}
    </div>
  );
}
