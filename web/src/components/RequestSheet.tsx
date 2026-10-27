import { useEffect, useRef, useState } from "react";
import { api } from "../api";
import { ago, amount, pcs, roleLabels, useLabels, useMeta, useUser } from "../meta";
import type { Listing } from "../types";
import { Gallery, RouteChip } from "./bits";
import { ShareButton } from "./share";

type Props = {
  listing: Listing;
  onClose: () => void;
  onRequested: (message: string) => void;
};

// RequestSheet is the dialog a seeker uses to request a listing. The request
// goes out under their account; Escape or a click outside closes it.
export default function RequestSheet({ listing, onClose, onRequested }: Props) {
  const meta = useMeta();
  const user = useUser();
  const labels = useLabels();
  const [use, setUse] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const first = useRef<HTMLInputElement>(null);

  const route = meta.routes.find((r) => r.id === listing.route);
  // Mirrors the server rule so the user sees why before submitting.
  const blocked = route?.recyclersOnly && user.role !== "recycler";

  useEffect(() => {
    first.current?.focus();
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api.request({ listingId: listing.id, intendedUse: use });
      onRequested(`Request sent to ${listing.providerName}.`);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="sheet-backdrop" onClick={onClose}>
      <section className="sheet" role="dialog" aria-modal="true" aria-labelledby="sheet-title" onClick={(e) => e.stopPropagation()}>
        <header className="sheet-head">
          <div>
            <p className="eyebrow">{labels.material(listing.material)}</p>
            <h2 id="sheet-title">{listing.item}</h2>
          </div>
          <div className="sheet-tools">
            <ShareButton listing={listing} />
            <button className="icon-btn" onClick={onClose} aria-label="Close">×</button>
          </div>
        </header>

        <Gallery id={listing.id} photos={listing.photos} item={listing.item} />

        <dl className="facts">
          <div><dt>Amount</dt><dd>{listing.quantity && listing.quantity !== pcs(listing.pieces) && <>{listing.quantity} <span className="muted">·</span></>} <span className="muted">{amount(listing)}</span></dd></div>
          <div><dt>Condition</dt><dd>{listing.condition || "—"}</dd></div>
          <div><dt>From</dt><dd>{listing.providerName}, {listing.area}</dd></div>
          <div><dt>Route</dt><dd><RouteChip route={listing.route} /></dd></div>
          {listing.notes && <div><dt>Notes</dt><dd>{listing.notes}</dd></div>}
          <div><dt>Listed</dt><dd>{ago(listing.createdAt)}</dd></div>
        </dl>

        <form className="form" onSubmit={submit}>
          <div className="field">
            <label htmlFor="req-use">What will you use it for?</label>
            <input id="req-use" ref={first} value={use} onChange={(e) => setUse(e.target.value)} maxLength={140} placeholder="e.g. stools for a class project" />
            <span className="fine">Requesting as {user.name} ({roleLabels[user.role].toLowerCase()}). The giver sees this and decides.</span>
          </div>
          {blocked && (
            <p className="banner banner-warn">This material needs specialised handling, so only recyclers can collect it.</p>
          )}
          {error && <p className="banner banner-error">{error}</p>}
          <button className="btn btn-primary" disabled={busy || blocked}>
            {busy ? "Sending…" : "Request this material"}
          </button>
        </form>
      </section>
    </div>
  );
}
