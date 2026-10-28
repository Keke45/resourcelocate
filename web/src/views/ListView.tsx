import { useEffect, useRef, useState } from "react";
import { api, photoUrl, shrinkPhoto, type NewListing } from "../api";
import { pcs, useMeta, useUser } from "../meta";
import type { Listing } from "../types";

// With editing set, the form changes that listing instead of publishing a
// new one; onCancel leaves without saving.
type Props = { editing?: Listing | null; onChanged: (message: string, next?: "dashboard") => void; onCancel?: () => void };

const blank: NewListing = {
  contact: "", item: "", material: "", quantity: "", pieces: 0, estKg: 0,
  condition: "", route: "reuse", area: "", notes: "",
};

// A photo picked for the listing, already shrunk for mobile data, with a
// local preview URL.
type Picked = { blob: Blob; url: string };

// ListView is the giver side: describe the material, optionally add a few
// reference photos so people looking can see what it is, and publish. It
// also edits an open listing, photos included.
export default function ListView({ editing, onChanged, onCancel }: Props) {
  const meta = useMeta();
  const user = useUser();
  // Pickup area starts at the user's own area.
  const fresh = (): NewListing => ({ ...blank, area: user.area });
  const [form, setForm] = useState<NewListing>(() => {
    if (!editing) return fresh();
    const { contact, item, material, quantity, pieces, estKg, condition, route, area, notes } = editing;
    return { contact, item, material, quantity, pieces, estKg, condition, route, area, notes };
  });
  // Loose material (offcuts, sawdust) is listed by weight; separate items
  // (monitors, desks) by count.
  const [measure, setMeasure] = useState<"weight" | "count">(editing && editing.pieces > 0 ? "count" : "weight");
  const [kgText, setKgText] = useState(editing?.estKg ? String(editing.estKg) : "");
  const [piecesText, setPiecesText] = useState(editing?.pieces ? String(editing.pieces) : "");
  // When editing, the photos already on the listing (by number) that the
  // giver is keeping; the rest are removed on save.
  const [kept, setKept] = useState<number[]>(() => Array.from({ length: editing?.photos ?? 0 }, (_, i) => i));
  const [photos, setPhotos] = useState<Picked[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  // Free the preview URLs when the page closes.
  const current = useRef(photos);
  current.current = photos;
  useEffect(() => () => current.current.forEach((p) => URL.revokeObjectURL(p.url)), []);

  const set = <K extends keyof NewListing>(k: K, v: NewListing[K]) => setForm((f) => ({ ...f, [k]: v }));

  async function addPhotos(files: FileList) {
    const room = meta.maxPhotos - kept.length - photos.length;
    const picked = await Promise.all(
      Array.from(files).slice(0, room).map(async (f) => {
        const blob = await shrinkPhoto(f);
        return { blob, url: URL.createObjectURL(blob) };
      }),
    );
    setPhotos((ps) => [...ps, ...picked]);
  }

  function removePhoto(i: number) {
    URL.revokeObjectURL(photos[i].url);
    setPhotos((ps) => ps.filter((_, j) => j !== i));
  }

  const pieces = measure === "count" ? Number(piecesText) || 0 : 0;
  const estKg = measure === "weight" ? Number(kgText) || 0 : 0;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    const details = {
      ...form, quantity: measure === "count" ? pcs(pieces) : form.quantity, pieces, estKg: Math.round(estKg * 10) / 10,
    };
    try {
      if (editing) {
        await api.updateListing(editing.id, details);
        // Remove from the highest number down, so the photos still to go
        // keep their numbers as earlier ones are taken out.
        const removed = Array.from({ length: editing.photos }, (_, i) => i).filter((i) => !kept.includes(i)).reverse();
        let failed = 0;
        for (const n of removed) await api.deletePhoto(editing.id, n).catch(() => failed++);
        for (const p of photos) await api.addPhoto(editing.id, p.blob).catch(() => failed++);
        photos.forEach((p) => URL.revokeObjectURL(p.url));
        setPhotos([]);
        onChanged(`"${details.item}" is updated.${failed ? " Some photo changes didn't go through." : ""}`, "dashboard");
        return;
      }
      const { listing: l, waiting } = await api.createListing(details);
      // The listing is live even if a photo fails; say so rather than fail the whole publish.
      let failed = 0;
      for (const p of photos) await api.addPhoto(l.id, p.blob).catch(() => failed++);
      setForm(fresh());
      setKgText("");
      setPiecesText("");
      setMeasure("weight");
      photos.forEach((p) => URL.revokeObjectURL(p.url));
      setPhotos([]);
      const lookers = waiting > 0 ? ` ${waiting} ${waiting === 1 ? "person is" : "people are"} looking for this material.` : "";
      const photoNote = failed ? ` ${failed === 1 ? "A photo" : `${failed} photos`} couldn't be uploaded.` : "";
      onChanged(`"${l.item}" is listed.${lookers}${photoNote}`, "dashboard");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="give">
      <header className="view-head">
        <h1 className="view-title">{editing ? "Edit your listing" : "Give away unwanted material"}</h1>
        <p className="view-lede">{editing
          ? "Changes show straight away to anyone looking. Anyone who already asked for it keeps their request."
          : "Describe what you have. Once it's live, students, makers and recyclers nearby can find it and ask for it."}</p>
      </header>

        <form className="panel form give-form" onSubmit={submit} aria-label="Listing details">
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
              <select id="l-route" value={form.route} onChange={(e) => set("route", e.target.value)}>
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
            <div className="field">
              <label htmlFor="l-pieces">Number of pieces</label>
              <input id="l-pieces" inputMode="numeric" value={piecesText} required placeholder="e.g. 12"
                onChange={(e) => setPiecesText(e.target.value.replace(/[^0-9]/g, ""))} />
            </div>
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
          <fieldset className="field">
            <legend>Photos (optional)</legend>
            <p className="fine">Up to {meta.maxPhotos}. They help people see what you're giving away before they ask for it.</p>
            <div className="photo-picks">
              {editing && kept.map((n) => (
                <div key={`kept-${n}`} className="photo-pick">
                  <img src={photoUrl(editing.id, n)} alt={`Photo ${n + 1}`} />
                  <button type="button" className="photo-remove" onClick={() => setKept((k) => k.filter((x) => x !== n))} aria-label={`Remove photo ${n + 1}`}>×</button>
                </div>
              ))}
              {photos.map((p, i) => (
                <div key={p.url} className="photo-pick">
                  <img src={p.url} alt={`Photo ${i + 1}`} />
                  <button type="button" className="photo-remove" onClick={() => removePhoto(i)} aria-label={`Remove photo ${i + 1}`}>×</button>
                </div>
              ))}
              {kept.length + photos.length < meta.maxPhotos && (
                <label className="photo-add" htmlFor="photo-input">
                  <span aria-hidden="true">+</span>
                  Add photo
                </label>
              )}
            </div>
            <input id="photo-input" type="file" accept="image/jpeg,image/png,image/webp" multiple className="visually-hidden"
              onChange={(e) => { if (e.target.files?.length) addPhotos(e.target.files); e.target.value = ""; }} />
          </fieldset>
          <p className="fine">Hazardous, medical and chemical waste can't be listed.</p>
          {error && <p className="banner banner-error">{error}</p>}
          <div className="actions">
            <button className="btn btn-primary" disabled={busy}>
              {editing ? (busy ? "Saving…" : "Save changes") : busy ? "Publishing…" : "Publish listing"}
            </button>
            {editing && <button type="button" className="btn" onClick={onCancel} disabled={busy}>Cancel</button>}
          </div>
        </form>
    </div>
  );
}
