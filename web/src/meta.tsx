import { createContext, useContext } from "react";
import type { Meta, User } from "./types";

// The catalog (materials, routes, areas...) is loaded once in App and shared
// through this context so every view uses the same labels.
export const MetaContext = createContext<Meta | null>(null);

export function useMeta(): Meta {
  const m = useContext(MetaContext);
  if (!m) throw new Error("useMeta outside MetaContext");
  return m;
}

// useLabels turns catalog ids such as "wood" into display labels, falling
// back to the id if the catalog has no entry.
// The signed-in user, provided by App around every signed-in screen.
export const UserContext = createContext<User | null>(null);

export function useUser(): User {
  const u = useContext(UserContext);
  if (!u) throw new Error("useUser outside UserContext");
  return u;
}

export function useLabels() {
  const meta = useMeta();
  const find = <T extends { id: string; label: string }>(list: T[], id: string) =>
    list.find((x) => x.id === id)?.label ?? id;
  return {
    material: (id: string) => find(meta.materials, id),
    route: (id: string) => find(meta.routes, id),
    outcome: (id: string) => find(meta.outcomes, id),
  };
}

export const roleLabels: Record<string, string> = {
  individual: "Student or individual",
  business: "Business or group",
  recycler: "Recycler",
};

// kg formats a weight, switching to tonnes from 10,000 kg so smaller
// totals keep their precision.
export function kg(n: number): string {
  if (n >= 10000) return `${(n / 1000).toLocaleString(undefined, { maximumFractionDigits: 1 })} t`;
  return `${n.toLocaleString(undefined, { maximumFractionDigits: 1 })} kg`;
}

// pcs formats a count of items: "1 piece", "12 pieces".
export function pcs(n: number): string {
  return `${n.toLocaleString()} ${n === 1 ? "piece" : "pieces"}`;
}

// amount describes how much a listing offers, in its own unit.
export function amount(l: { pieces: number; estKg: number }): string {
  return l.pieces > 0 ? pcs(l.pieces) : `about ${kg(l.estKg)}`;
}

// handed describes an Amount (what was given or claimed): "45 kg and 10
// pieces", "10 pieces", or "nothing yet".
export function handed(a: { kg: number; pieces: number }): string {
  const parts = [a.kg > 0 && kg(a.kg), a.pieces > 0 && pcs(a.pieces)].filter(Boolean);
  return parts.length ? parts.join(" and ") : "nothing yet";
}

// needAmount describes how much a need asks for, or "any amount".
export function needAmount(n: { kg: number; pieces: number }): string {
  if (n.pieces > 0) return pcs(n.pieces);
  if (n.kg > 0) return kg(n.kg);
  return "any amount";
}

// ago gives a short relative time such as "5 h ago" for an ISO timestamp.
export function ago(iso: string): string {
  const mins = Math.max(0, Math.round((Date.now() - Date.parse(iso)) / 60000));
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins} min ago`;
  const h = Math.round(mins / 60);
  if (h < 24) return `${h} h ago`;
  const d = Math.round(h / 24);
  return d === 1 ? "yesterday" : `${d} days ago`;
}
