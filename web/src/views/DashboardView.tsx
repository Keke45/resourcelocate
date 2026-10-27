import { useEffect, useState } from "react";
import { api, photoUrl } from "../api";
import { ago, amount, handed, needAmount, useLabels, useUser } from "../meta";
import type { Dashboard, Listing, NeedView, Transfer } from "../types";
import { RouteChip } from "../components/bits";
import { ShareButton } from "../components/share";
import RequestSheet from "../components/RequestSheet";

type Tab = "dashboard" | "find" | "give" | "handovers";
type Props = { version: number; onChanged: (message: string, next?: Tab) => void; go: (t: Tab) => void };

// myTurn reports whether a hand-over is waiting on the signed-in user: the
// giver answers requests and confirms pickups, the receiver records outcomes.
export function myTurn(t: Transfer): boolean {
  if (t.side === "giving") return t.status === "requested" || t.status === "accepted";
  return t.status === "collected";
}

const waitingOnOthers = (t: Transfer) =>
  !myTurn(t) && (t.status === "requested" || t.status === "accepted" || t.status === "collected");

const nextStep: Record<string, string> = {
  "giving-requested": "Accept or decline this request",
  "giving-accepted": "Confirm the hand-over once it's collected",
  "receiving-collected": "Say what you did with it",
  "receiving-requested": "Waiting for the giver to answer",
  "receiving-accepted": "Accepted. Arrange the pickup",
  "giving-collected": "Waiting for the receiver to record the outcome",
};

const listingStatus: Record<Listing["status"], string> = {
  open: "Open",
  reserved: "Reserved",
  closed: "Handed over",
  withdrawn: "Withdrawn",
};

// DashboardView is the signed-in home: what needs the user's attention,
// their saved needs with live matches, their listings, and how much they
// have given and claimed.
export default function DashboardView({ version, onChanged, go }: Props) {
  const user = useUser();
  const [data, setData] = useState<Dashboard | null>(null);
  const [error, setError] = useState("");
  const [requesting, setRequesting] = useState<Listing | null>(null);

  useEffect(() => {
    api.dashboard().then(setData, (e: Error) => setError(e.message));
  }, [version]);

  if (error) return <p className="banner banner-error">{error}</p>;
  if (!data) return <p className="loading">Loading your dashboard…</p>;

  const todo = data.transfers.filter(myTurn);
  const waiting = data.transfers.filter(waitingOnOthers);
  const openListings = data.listings.filter((l) => l.status === "open").length;
  const { given, claimed } = data.totals;
  const done = given.handOvers + claimed.handOvers;
  const newcomer = data.listings.length === 0 && data.needs.length === 0 && data.transfers.length === 0;

  return (
    <div className="dashboard">
      <header className="dash-head">
        <div>
          <h1 className="view-title">Hello, {user.name}</h1>
          <p className="view-lede">{user.area} · {done
            ? `${done} finished ${done === 1 ? "hand-over" : "hand-overs"} so far`
            : "Let's get your first hand-over going"}</p>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => go("give")}>Give something away</button>
          <button className="btn" onClick={() => go("find")}>Find materials</button>
        </div>
      </header>

      {newcomer && (
        <section className="panel welcome">
          <h2 className="section-title">Two ways to start</h2>
          <ul className="welcome-list">
            <li><strong>Have something you don't need?</strong> List it on <button className="link" onClick={() => go("give")}>Give</button>. It takes a minute.</li>
            <li><strong>Looking for something?</strong> Search on <button className="link" onClick={() => go("find")}>Find</button> and save it as a need. New matches will show up here.</li>
          </ul>
        </section>
      )}

      <section className="stats" aria-label="What you've given and claimed">
        <div className="stat stat-lead">
          <p className="stat-label">You've given</p>
          <p className="stat-value num">{handed(given)}</p>
          <p className="stat-note">in {given.handOvers} finished {given.handOvers === 1 ? "hand-over" : "hand-overs"}</p>
        </div>
        <div className="stat">
          <p className="stat-label">You've claimed</p>
          <p className="stat-value num">{handed(claimed)}</p>
          <p className="stat-note">in {claimed.handOvers} finished {claimed.handOvers === 1 ? "hand-over" : "hand-overs"}</p>
        </div>
        <div className="stat">
          <p className="stat-label">In progress</p>
          <p className="stat-value num">{data.totals.inProgress}</p>
          <p className="stat-note">requested, accepted or picked up</p>
        </div>
        <div className="stat">
          <p className="stat-label">Open listings</p>
          <p className="stat-value num">{openListings}</p>
          <p className="stat-note">{data.needs.length} saved {data.needs.length === 1 ? "need" : "needs"}</p>
        </div>
      </section>

      <div className="dash-grid">
        <section className="panel dash-section" aria-labelledby="todo-title">
          <h2 id="todo-title" className="section-title">Needs your attention {todo.length > 0 && <span className="count">{todo.length}</span>}</h2>
          {todo.length === 0 ? <p className="empty-inline">Nothing waiting on you right now.</p> : (
            <ul className="mini-list">
              {todo.map((t) => <TransferRow key={t.id} t={t} onOpen={() => go("handovers")} />)}
            </ul>
          )}
          {waiting.length > 0 && (
            <>
              <h3 className="sub-title">Waiting on others</h3>
              <ul className="mini-list">
                {waiting.map((t) => <TransferRow key={t.id} t={t} onOpen={() => go("handovers")} />)}
              </ul>
            </>
          )}
        </section>

        <section className="panel dash-section" aria-labelledby="needs-title">
          <div className="section-row">
            <h2 id="needs-title" className="section-title">What you're looking for</h2>
            <button className="link" onClick={() => go("find")}>Add a need</button>
          </div>
          {data.needs.length === 0 ? (
            <p className="empty-inline">No saved needs. Search on Find and save it, and matches will appear here as people list them.</p>
          ) : (
            <ul className="mini-list">
              {data.needs.map((n) => (
                <NeedCard key={n.id} n={n} onRequest={setRequesting}
                  onClose={async () => {
                    try {
                      await api.closeNeed(n.id);
                      onChanged("Need removed.");
                    } catch (e) {
                      setError((e as Error).message);
                    }
                  }} />
              ))}
            </ul>
          )}
        </section>

        <section className="panel dash-section dash-wide" aria-labelledby="listings-title">
          <div className="section-row">
            <h2 id="listings-title" className="section-title">Your listings</h2>
            <button className="link" onClick={() => go("give")}>List something</button>
          </div>
          {data.listings.length === 0 ? <p className="empty-inline">You haven't listed anything yet.</p> : (
            <ul className="mini-list">
              {data.listings.map((l) => {
                const requests = data.transfers.filter((t) => t.listingId === l.id && t.status === "requested").length;
                return (
                  <li key={l.id} className="mini-row">
                    {l.photos > 0 && <img className="thumb thumb-small" src={photoUrl(l.id)} alt="" loading="lazy" />}
                    <div className="mini-main">
                      <p className="mini-title">{l.item}</p>
                      <p className="listing-meta">
                        <span className="num">{amount(l)}</span> · {l.area} · listed {ago(l.createdAt)}
                        {requests > 0 && <> · <strong>{requests} {requests === 1 ? "request" : "requests"}</strong></>}
                      </p>
                    </div>
                    <div className="mini-side">
                      <RouteChip route={l.route} />
                      <span className={`chip status-chip status-${l.status}`}>{listingStatus[l.status]}</span>
                      {l.status === "open" && <ShareButton listing={l} />}
                      {l.status === "open" && (
                        <button className="btn btn-small" onClick={async () => {
                          if (!confirm(`Withdraw "${l.item}"? Anyone who requested it will be told it's no longer available.`)) return;
                          try {
                            await api.withdrawListing(l.id);
                            onChanged("Listing withdrawn.");
                          } catch (e) {
                            setError((e as Error).message);
                          }
                        }}>Withdraw</button>
                      )}
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </section>
      </div>

      {requesting && (
        <RequestSheet listing={requesting} onClose={() => setRequesting(null)}
          onRequested={(msg) => { setRequesting(null); onChanged(msg, "handovers"); }} />
      )}
    </div>
  );
}

function TransferRow({ t, onOpen }: { t: Transfer; onOpen: () => void }) {
  const other = t.side === "giving" ? t.receiverName : t.providerName;
  return (
    <li className="mini-row">
      <div className="mini-main">
        <p className="mini-title">{t.item}</p>
        <p className="listing-meta">
          {t.side === "giving" ? "To" : "From"} <strong>{other}</strong> · {nextStep[`${t.side}-${t.status}`]}
        </p>
      </div>
      <div className="mini-side">
        <span className="when">{ago(t.updatedAt)}</span>
        <button className="btn btn-small" onClick={onOpen}>Open</button>
      </div>
    </li>
  );
}

// NeedCard shows a saved need and its best current matches.
function NeedCard({ n, onRequest, onClose }: { n: NeedView; onRequest: (l: Listing) => void; onClose: () => void }) {
  const labels = useLabels();
  const top = n.matches.slice(0, 3);
  return (
    <li className="need-card">
      <div className="mini-row">
        <div className="mini-main">
          <p className="mini-title">{labels.material(n.material)}, {needAmount(n)}</p>
          <p className="listing-meta">
            {n.area || "Anywhere"} · saved {ago(n.createdAt)}{n.note && ` · ${n.note}`}
          </p>
        </div>
        <div className="mini-side">
          <span className={`chip ${n.matches.length ? "chip-good" : "chip-quiet"}`}>
            {n.matches.length ? `${n.matches.length} ${n.matches.length === 1 ? "match" : "matches"}` : "No matches yet"}
          </span>
          <button className="btn btn-small" onClick={onClose} aria-label="Remove this need">Remove</button>
        </div>
      </div>
      {top.length > 0 && (
        <ul className="need-matches">
          {top.map((m) => (
            <li key={m.listing.id}>
              <button className="match-row" onClick={() => onRequest(m.listing)}>
                <span>{m.listing.item}</span>
                <span className="muted">
                  {amount(m.listing)} · {m.listing.area}
                </span>
                <span className="num fit-score">{m.score}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </li>
  );
}
