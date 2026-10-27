import { useEffect, useState } from "react";
import { api, type Step } from "../api";
import { ago, kg, pcs, roleLabels, useLabels, useMeta } from "../meta";
import type { Transfer } from "../types";
import { RouteChip, Stepper } from "../components/bits";

type Props = { version: number; onChanged: (message: string) => void };
type Filter = "active" | "done" | "all";

// Statuses that still need someone to act.
const active = new Set(["requested", "accepted", "collected"]);

// ActivityView lists the signed-in user's hand-overs, as giver or receiver,
// and shows each side only the actions that are theirs. The server checks
// the same rules.
export default function ActivityView({ version, onChanged }: Props) {
  const [transfers, setTransfers] = useState<Transfer[] | null>(null);
  const [error, setError] = useState("");
  const [filter, setFilter] = useState<Filter>("active");

  useEffect(() => {
    api.transfers().then(setTransfers, (e: Error) => setError(e.message));
  }, [version]);

  const shown = transfers?.filter((t) =>
    filter === "all" ? true : filter === "active" ? active.has(t.status) : !active.has(t.status),
  );

  return (
    <div className="activity">
      <header className="view-head">
        <h1 className="view-title">Hand-overs</h1>
        <p className="view-lede">
          Everything you're giving or receiving. The giver answers requests and confirms the pickup; the receiver records what happened to the material.
        </p>
      </header>
      <div className="segmented" role="tablist" aria-label="Filter hand-overs">
        {(["active", "done", "all"] as Filter[]).map((f) => (
          <button key={f} role="tab" aria-selected={filter === f} onClick={() => setFilter(f)}>
            {f === "active" ? "In progress" : f === "done" ? "Finished" : "All"}
          </button>
        ))}
      </div>
      {error && <p className="banner banner-error">{error}</p>}
      {shown?.length === 0 && (
        <p className="empty">
          {filter === "active" ? "No hand-overs in progress. Request something on Find, or list something on Give." : "Nothing here yet."}
        </p>
      )}
      <ul className="cards">
        {shown?.map((t) => <TransferCard key={t.id} t={t} onChanged={onChanged} />)}
      </ul>
    </div>
  );
}

// TransferCard shows one hand-over, its progress, and the next action.
// Collecting records how much changed hands, in the listing's own unit.
function TransferCard({ t, onChanged }: { t: Transfer; onChanged: (m: string) => void }) {
  const meta = useMeta();
  const labels = useLabels();
  const [form, setForm] = useState<"" | "collect" | "outcome">("");
  const [weight, setWeight] = useState("");
  const [pieces, setPieces] = useState("");
  const [outcome, setOutcome] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function run(step: Step, message: string) {
    setBusy(true);
    setError("");
    try {
      await api.advance(t.id, step);
      setForm("");
      onChanged(message);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  // Counted listings ask for pieces, bulk listings for kg.
  const counted = t.listingPieces > 0;

  return (
    <li className={`card transfer status-${t.status}`}>
      <div className="transfer-head">
        <div>
          <p className="eyebrow">{labels.material(t.material)} · {t.area}</p>
          <h3 className="listing-item">{t.item}</h3>
          <p className="listing-meta">
            {t.providerName} → <strong>{t.receiverName}</strong>
            <span className="muted"> ({roleLabels[t.receiverRole]})</span>
          </p>
          {t.intendedUse && <p className="listing-meta muted">For: {t.intendedUse}</p>}
        </div>
        <div className="transfer-tags">
          <RouteChip route={t.route} />
          <span className={`chip ${t.side === "giving" ? "chip-giving" : "chip-receiving"}`}>
            {t.side === "giving" ? "You're giving" : "You're receiving"}
          </span>
        </div>
      </div>

      <Stepper status={t.status} />

      {(t.status === "collected" || t.status === "completed") && (
        <p className="ledger">
          Handed over: <span className="num">{counted ? pcs(t.confirmedPieces) : kg(t.confirmedKg)}</span>.
          {t.status === "completed" && <> Outcome: {labels.outcome(t.outcome).toLowerCase()}{t.outcomeNote && ` (${t.outcomeNote})`}.</>}
        </p>
      )}

      {t.contact && t.side === "receiving" && t.status === "accepted" && (
        <p className="banner banner-info">Accepted. Contact {t.providerName} on <strong>{t.contact}</strong> to arrange the pickup.</p>
      )}
      {t.side === "receiving" && t.status === "accepted" && !t.contact && (
        <p className="banner banner-info">Accepted. {t.providerName} will confirm once you've collected it.</p>
      )}
      {t.side === "receiving" && t.status === "requested" && (
        <p className="fine">Waiting for {t.providerName} to answer your request.</p>
      )}
      {t.side === "giving" && t.status === "collected" && (
        <p className="fine">Waiting for {t.receiverName} to say what they did with it.</p>
      )}

      {t.side === "giving" && t.status === "requested" && (
        <div className="actions">
          <button className="btn btn-primary" disabled={busy}
            onClick={() => run({ action: "accept" }, `Accepted. Arrange pickup with ${t.receiverName}.`)}>Accept request</button>
          <button className="btn" disabled={busy} onClick={() => run({ action: "decline" }, "Request declined.")}>Decline</button>
        </div>
      )}

      {t.side === "giving" && t.status === "accepted" && form === "" && (
        <div className="actions">
          <button className="btn btn-primary" onClick={() => setForm("collect")}>Confirm hand-over</button>
          <button className="btn" disabled={busy}
            onClick={() => run({ action: "fail" }, "Marked as not collected. The listing is open again.")}>Not collected</button>
        </div>
      )}

      {form === "collect" && (
        <form className="inline-form" onSubmit={(e) => {
          e.preventDefault();
          run({ action: "collect", confirmedKg: counted ? 0 : Number(weight) || 0, pieces: counted ? Number(pieces) || 0 : 0 }, "Hand-over recorded.");
        }}>
          {counted ? (
            <div className="field">
              <label htmlFor={`p-${t.id}`}>Pieces handed over (of {t.listingPieces})</label>
              <input id={`p-${t.id}`} inputMode="numeric" required value={pieces}
                onChange={(e) => setPieces(e.target.value.replace(/[^0-9]/g, ""))} />
            </div>
          ) : (
            <div className="field">
              <label htmlFor={`w-${t.id}`}>Weight handed over (kg)</label>
              <input id={`w-${t.id}`} inputMode="decimal" required value={weight}
                placeholder={`Listed as about ${t.listingEstKg} kg`}
                onChange={(e) => setWeight(e.target.value.replace(/[^0-9.]/g, ""))} />
            </div>
          )}
          <div className="actions">
            <button className="btn btn-primary" disabled={busy || (counted ? !pieces : !weight)}>Save hand-over</button>
            <button type="button" className="btn" onClick={() => setForm("")}>Cancel</button>
          </div>
        </form>
      )}

      {t.side === "receiving" && t.status === "collected" && form === "" && (
        <div className="actions">
          <button className="btn btn-primary" onClick={() => setForm("outcome")}>Record what happened</button>
        </div>
      )}

      {form === "outcome" && (
        <form className="inline-form" onSubmit={(e) => {
          e.preventDefault();
          run({ action: "outcome", outcome, outcomeNote: note }, "Outcome recorded. The hand-over is finished.");
        }}>
          <fieldset className="field">
            <legend>What happened to it?</legend>
            {meta.outcomes.map((o) => (
              <label key={o.id} className="radio">
                <input type="radio" name={`o-${t.id}`} value={o.id} checked={outcome === o.id} onChange={() => setOutcome(o.id)} required />
                {o.label}
              </label>
            ))}
          </fieldset>
          <div className="field">
            <label htmlFor={`n-${t.id}`}>Short note (optional)</label>
            <input id={`n-${t.id}`} value={note} onChange={(e) => setNote(e.target.value)} maxLength={140} placeholder="e.g. made 3 stools" />
          </div>
          <div className="actions">
            <button className="btn btn-primary" disabled={busy || !outcome}>Save outcome</button>
            <button type="button" className="btn" onClick={() => setForm("")}>Cancel</button>
          </div>
        </form>
      )}

      {error && <p className="banner banner-error">{error}</p>}
      <p className="when">Updated {ago(t.updatedAt)}</p>
    </li>
  );
}
