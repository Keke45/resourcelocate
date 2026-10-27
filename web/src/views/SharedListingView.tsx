import { useEffect, useState } from "react";
import { api } from "../api";
import { UserContext, ago, amount, useLabels } from "../meta";
import type { Listing, User } from "../types";
import { Gallery, RouteChip } from "../components/bits";
import { ShareButton } from "../components/share";
import RequestSheet from "../components/RequestSheet";

type Props = {
  id: string;
  user: User | null;
  // Signed out: log in or sign up, then come back to this listing.
  onLogin: () => void;
  onSignup: () => void;
  onRequested: (message: string) => void;
  onBrowse: () => void;
};

// SharedListingView is the page a shared link (/l/{id}) opens. Anyone can
// see the listing; signing in lets them ask for it.
export default function SharedListingView({ id, user, onLogin, onSignup, onRequested, onBrowse }: Props) {
  const labels = useLabels();
  const [listing, setListing] = useState<Listing | null>(null);
  const [error, setError] = useState("");
  const [requesting, setRequesting] = useState(false);

  useEffect(() => {
    api.publicListing(id).then(setListing, (e: Error) => setError(e.message));
  }, [id]);

  if (error) {
    return (
      <section className="panel shared">
        <h1 className="view-title">This listing isn't available</h1>
        <p className="view-lede">{error}</p>
        <div className="actions"><button className="btn btn-primary" onClick={onBrowse}>See what else is available</button></div>
      </section>
    );
  }
  if (!listing) return <p className="loading">Loading…</p>;

  const own = user?.id === listing.ownerId;
  const open = listing.status === "open";

  return (
    <article className="panel shared" aria-labelledby="shared-title">
      <p className="eyebrow">{labels.material(listing.material)} · free to a good home</p>
      <h1 id="shared-title" className="view-title">{listing.item}</h1>
      <Gallery id={listing.id} photos={listing.photos} item={listing.item} />
      <dl className="facts">
        <div><dt>Amount</dt><dd>{amount(listing)}</dd></div>
        {listing.condition && <div><dt>Condition</dt><dd>{listing.condition}</dd></div>}
        <div><dt>From</dt><dd>{listing.providerName}, {listing.area}</dd></div>
        <div><dt>Route</dt><dd><RouteChip route={listing.route} /></dd></div>
        {listing.notes && <div><dt>Notes</dt><dd>{listing.notes}</dd></div>}
        <div><dt>Listed</dt><dd>{ago(listing.createdAt)}</dd></div>
      </dl>

      {!open && <p className="banner banner-info">This has already been promised to someone.</p>}
      {own && <p className="banner banner-info">This is your listing. Share the link anywhere so people nearby can find it.</p>}

      <div className="actions">
        {open && !own && user && <button className="btn btn-primary" onClick={() => setRequesting(true)}>Ask for it</button>}
        {open && !user && (
          <>
            <button className="btn btn-primary" onClick={onLogin}>Log in to ask for it</button>
            <button className="btn" onClick={onSignup}>Create a free account</button>
          </>
        )}
        <ShareButton listing={listing} className="btn" />
        {!open && <button className="btn" onClick={onBrowse}>See what else is available</button>}
      </div>
      {!user && open && <p className="fine">It takes a minute to sign up. The giver decides who gets it.</p>}

      {requesting && user && (
        <UserContext.Provider value={user}>
          <RequestSheet listing={listing} onClose={() => setRequesting(false)}
            onRequested={(msg) => { setRequesting(false); onRequested(msg); }} />
        </UserContext.Provider>
      )}
    </article>
  );
}
