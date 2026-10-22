import { useEffect, useState } from "react";
import { api, type NewListing } from "../api";
import { pcs, useMeta, useUser } from "../meta";
import type { Draft } from "../types";

type Props = { onChanged: (message: string, next?: "dashboard") => void };

const blank: NewListing = {
  contact: "", item: "", material: "", quantity: "", pieces: 0, estKg: 0,
  condition: "", route: "reuse", area: "", notes: "",
};

// ListView is the provider side: an optional photo pre-fills the form, the
// provider checks every field, then publishes the listing.
export default function ListView({ onChanged }: Props) {
  const meta = useMeta();
  const user = useUser();
  // Pickup area starts at the user's own area.
  const fresh = (): NewListing => ({ ...blank, area: user.area });
  const [form, setForm] = useState<NewListing>(fresh);
  // Loose material (offcuts, sawdust) is listed by weight. Separate items
  // (monitors, desks) are listed by count plus an estimated weight per piece,
  // because impact is still calculated by weight.
  const [measure, setMeasure] = useState<"weight" | "count">("weight");
  const [kgText, setKgText] = useState("");
  const [piecesText, setPiecesText] = useState("");
  const [perPieceText, setPerPieceText] = useState("");
  const [photo, setPhoto] = useState<File | null>(null);
  const [preview, setPreview] = useState("");
  const [reading, setReading] = useState(false);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [photoNote, setPhotoNote] = useState("");
  const [notHazard, setNotHazard] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!photo) return;
    const url = URL.createObjectURL(photo);
    setPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [photo]);

  const set = <K extends keyof NewListing>(k: K, v: NewListing[K]) => setForm((f) => ({ ...f, [k]: v }));

  // readPhoto asks the server to draft a listing from the photo. The draft only
  // fills the form; nothing is published until the provider submits.
  async function readPhoto(file: File) {
    setPhoto(file);
    setDraft(null);
    setNotHazard(false);
    setPhotoNote("");
    if (!meta.photoReading) {
      setPhotoNote("Photo reading is off on this server, so fill in the details below.");
      return;
    }
    setReading(true);
    try {
      const d = await api.draftFromPhoto(file);
      if (!d.item) {
        setPhotoNote("We couldn't spot a listable material in that photo. Fill in the details below.");
        return;
      }
      setDraft(d);
      setForm((f) => ({ ...f, item: d.item, material: d.material, quantity: d.quantity, condition: d.condition, route: d.route }));
      const round = (n: number) => String(Math.round(n * 10) / 10);
      if (d.pieces > 0) {
        setMeasure("count");
        setPiecesText(String(d.pieces));
        setPerPieceText(d.estKg ? round(d.estKg / d.pieces) : "");
      } else {
        setMeasure("weight");
        setKgText(d.estKg ? round(d.estKg) : "");
      }
      setPhotoNote(
        d.confidence === "high"
          ? "Filled in from your photo. Check each field before you publish."
          : "Filled in from your photo, but it wasn't sure. Check the weight and material carefully.",
      );
    } catch (e) {
      setPhotoNote((e as Error).message);
    } finally {
      setReading(false);
    }
  }

  // A hazardous draft is locked to specialised handling (recyclers only)
  // unless the provider confirms it is not hazardous.
  const hazardBlock = draft?.hazardous && !notHazard;

  const pieces = measure === "count" ? Number(piecesText) || 0 : 0;
  const totalKg = measure === "count" ? pieces * (Number(perPieceText) || 0) : Number(kgText) || 0;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const { listing: l, waiting } = await api.createListing({ ...form, quantity: form.quantity || (pieces ? pcs(pieces) : ""), pieces, estKg: Math.round(totalKg * 10) / 10 });
      setForm(fresh());
      setKgText("");
      setPiecesText("");
      setPerPieceText("");
      setMeasure("weight");
      setPhoto(null);
      setPreview("");
      setDraft(null);
      setPhotoNote("");
      onChanged(
        waiting > 0
          ? `"${l.item}" is listed. ${waiting} ${waiting === 1 ? "person is" : "people are"} looking for this material.`
          : `"${l.item}" is listed.`,
        "dashboard",
      );
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="give">
      <header className="view-head">
        <h1 className="view-title">Give away unwanted material</h1>
        <p className="view-lede">Snap a photo and we'll draft the listing. You check it, then it goes live for students, makers and recyclers.</p>
      </header>

      <div className="give-grid">
        <section className="panel photo-panel" aria-labelledby="photo-title">
          <h2 id="photo-title" className="section-title">1. Photo</h2>
          <label className={`dropzone ${preview ? "has-photo" : ""}`} htmlFor="photo-input">
            {preview ? <img src={preview} alt="Your photo of the material" /> : (
              <span>
                <strong>Take or choose a photo</strong>
                <span className="muted">JPEG, PNG or WebP, under 8 MB</span>
              </span>
            )}
          </label>
          <input id="photo-input" type="file" accept="image/jpeg,image/png,image/webp,image/gif" capture="environment"
            className="visually-hidden"
            onChange={(e) => e.target.files?.[0] && readPhoto(e.target.files[0])} />
          {reading && <p className="banner banner-info">Reading your photo…</p>}
          {photoNote && !reading && <p className="banner banner-info">{photoNote}</p>}
          {draft?.hazardous && (
            <div className="banner banner-critical">
              <p><strong>This looks hazardous.</strong> {draft.hazardReason} ResourceLocate can't hand it to individuals. It's set to specialised handling so only recyclers will see it.</p>
              <label className="check">
                <input type="checkbox" checked={notHazard} onChange={(e) => setNotHazard(e.target.checked)} />
                I've checked and it isn't hazardous
              </label>
            </div>
          )}
          {!meta.photoReading && <p className="fine">Photo reading is off on this server. Every field can be filled in by hand.</p>}
        </section>

        <form className="panel form" onSubmit={submit} aria-labelledby="details-title">
          <h2 id="details-title" className="section-title">2. Details</h2>
          <div className="field">
            <label htmlFor="l-item">What is it?</label>
            <input id="l-item" value={form.item} onChange={(e) => set("item", e.target.value)} required maxLength={80} placeholder="e.g. Hardwood offcuts" />
          </div>
          <div className="row">
            <div className="field">
              <label htmlFor="l-material">Material</label>
              <select id="l-material" value={form.material} onChange={(e) => set("material", e.target.value)} required>
                <option value="" disabled>Choose…</option>
                {meta.materials.map((m) => <option key={m.id} value={m.id}>{m.label}</option>)}
              </select>
            </div>
            <div className="field">
              <label htmlFor="l-route">Where should it go?</label>
              <select id="l-route" value={hazardBlock ? "special" : form.route} disabled={hazardBlock}
                onChange={(e) => set("route", e.target.value)}>
                {meta.routes.map((r) => <option key={r.id} value={r.id}>{r.label}</option>)}
              </select>
            </div>
          </div>
          <fieldset className="field">
            <legend>How is it measured?</legend>
            <div className="segmented segmented-small" role="radiogroup">
              <button type="button" role="radio" aria-checked={measure === "weight"} onClick={() => setMeasure("weight")}>By weight</button>
              <button type="button" role="radio" aria-checked={measure === "count"} onClick={() => setMeasure("count")}>By count</button>
            </div>
          </fieldset>
          {measure === "weight" ? (
            <div className="row">
              <div className="field">
                <label htmlFor="l-qty">Quantity</label>
                <input id="l-qty" value={form.quantity} onChange={(e) => set("quantity", e.target.value)} maxLength={60} placeholder="e.g. 2 sacks" />
              </div>
              <div className="field">
                <label htmlFor="l-kg">Estimated weight (kg)</label>
                <input id="l-kg" inputMode="decimal" value={kgText} required
                  onChange={(e) => setKgText(e.target.value.replace(/[^0-9.]/g, ""))} />
              </div>
            </div>
          ) : (
            <>
              <div className="row">
                <div className="field">
                  <label htmlFor="l-pieces">Number of pieces</label>
                  <input id="l-pieces" inputMode="numeric" value={piecesText} required placeholder="e.g. 12"
                    onChange={(e) => setPiecesText(e.target.value.replace(/[^0-9]/g, ""))} />
                </div>
                <div className="field">
                  <label htmlFor="l-perpiece">Weight of one piece (kg, estimate)</label>
                  <input id="l-perpiece" inputMode="decimal" value={perPieceText} required placeholder="e.g. 4.5"
                    onChange={(e) => setPerPieceText(e.target.value.replace(/[^0-9.]/g, ""))} />
                </div>
              </div>
              <p className="fine">
                {pieces > 0 && totalKg > 0
                  ? `About ${Math.round(totalKg * 10) / 10} kg in total. Impact is calculated by weight, so a rough guess per piece is enough.`
                  : "Impact is calculated by weight, so add a rough guess per piece."}
              </p>
            </>
          )}
          <div className="field">
            <label htmlFor="l-condition">Condition</label>
            <input id="l-condition" value={form.condition} onChange={(e) => set("condition", e.target.value)} maxLength={80} placeholder="e.g. Clean, 20 to 60 cm pieces" />
          </div>
          <div className="row">
            <div className="field">
              <label htmlFor="l-area">Pickup area</label>
              <select id="l-area" value={form.area} onChange={(e) => set("area", e.target.value)} required>
                <option value="" disabled>Choose…</option>
                {meta.areas.map((a) => <option key={a}>{a}</option>)}
              </select>
            </div>
            <div className="field">
              <label htmlFor="l-contact">Phone or WhatsApp (optional)</label>
              <input id="l-contact" value={form.contact} onChange={(e) => set("contact", e.target.value)} maxLength={40} inputMode="tel" />
            </div>
          </div>
          <p className="fine">Listed as {user.name}. Your number is only shown to someone after you accept their request.</p>
          <div className="field">
            <label htmlFor="l-notes">Notes (optional)</label>
            <textarea id="l-notes" rows={2} value={form.notes} onChange={(e) => set("notes", e.target.value)} maxLength={280} />
          </div>
          <p className="fine">Hazardous, medical and chemical waste can't be listed.</p>
          {error && <p className="banner banner-error">{error}</p>}
          <button className="btn btn-primary" disabled={busy || reading}>
            {busy ? "Publishing…" : "Publish listing"}
          </button>
        </form>
      </div>
    </div>
  );
}
