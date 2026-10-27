import { useState } from "react";
import { photoUrl } from "../api";
import { useLabels } from "../meta";
import type { TransferStatus } from "../types";

export function RouteChip({ route }: { route: string }) {
  const labels = useLabels();
  return <span className={`chip route-${route}`}>{labels.route(route)}</span>;
}

const stages = ["Requested", "Accepted", "Collected", "Outcome"] as const;

const reached: Record<TransferStatus, number> = {
  requested: 1,
  accepted: 2,
  collected: 3,
  completed: 4,
  declined: 1,
  failed: 2,
};

// Stepper shows how far a transfer got; a declined or failed one stops in red.
export function Stepper({ status }: { status: TransferStatus }) {
  const n = reached[status];
  const stopped = status === "declined" || status === "failed";
  return (
    <ol className="stepper" aria-label={`Progress: ${status}`}>
      {stages.map((s, i) => {
        const state = i < n ? (stopped && i === n - 1 ? "stopped" : "done") : "todo";
        return (
          <li key={s} className={`step step-${state}`}>
            <span className="step-dot" aria-hidden="true" />
            <span className="step-name">
              {stopped && i === n - 1 ? (status === "declined" ? "Declined" : "Not collected") : s}
            </span>
          </li>
        );
      })}
    </ol>
  );
}

// Gallery shows a listing's photos: one large, with thumbnails to switch
// between them when there are several.
export function Gallery({ id, photos, item }: { id: string; photos: number; item: string }) {
  const [shown, setShown] = useState(0);
  if (photos === 0) return null;
  return (
    <div className="gallery">
      <img className="sheet-photo" src={photoUrl(id, shown)} alt={`Photo ${shown + 1} of ${item}`} />
      {photos > 1 && (
        <div className="gallery-thumbs">
          {Array.from({ length: photos }, (_, i) => (
            <button key={i} type="button" aria-pressed={i === shown} aria-label={`Show photo ${i + 1}`} onClick={() => setShown(i)}>
              <img src={photoUrl(id, i)} alt="" loading="lazy" />
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
