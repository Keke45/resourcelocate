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
