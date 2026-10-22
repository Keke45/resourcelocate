import type { Dashboard, Draft, Listing, Match, Meta, NeedView, SavedNeed, Stats, Transfer, User } from "./types";

// Thin client for the Go API. Every call rejects with an ApiError whose
// message is safe to show the user as is.

export class ApiError extends Error {
  constructor(message: string, readonly status = 0) {
    super(message);
  }
}

// onSignedOut is set by App so any 401 (an expired session) sends the user
// back to the login page.
let onSignedOut: () => void = () => {};
export function setSignedOutHandler(f: () => void) {
  onSignedOut = f;
}

// call fetches a path and returns the parsed JSON body, turning network
// failures and non-2xx answers into a readable ApiError.
async function call<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(path, { credentials: "same-origin", ...init });
  } catch {
    throw new ApiError("Can't reach the server. Check your connection and try again.");
  }
  const body = res.status === 204 ? null : await res.json().catch(() => null);
  if (!res.ok) {
    if (res.status === 401 && !path.startsWith("/api/auth/")) onSignedOut();
    throw new ApiError(body?.error ?? `The server answered ${res.status}.`, res.status);
  }
  return body as T;
}

const json = (method: string, body?: unknown): RequestInit => ({
  method,
  headers: { "Content-Type": "application/json" },
  body: body === undefined ? undefined : JSON.stringify(body),
});

export type NewListing = Omit<Listing, "id" | "ownerId" | "providerName" | "status" | "createdAt">;
export type NewNeed = Pick<SavedNeed, "material" | "kg" | "pieces" | "area" | "note">;
export type Signup = { name: string; email: string; password: string; role: string; area: string };

// One lifecycle step for a hand-over. Steps that record data carry it in the
// body; the others are a bare POST.
export type Step =
  | { action: "accept" | "decline" | "fail" }
  | { action: "collect"; confirmedKg: number; pieces: number; counterfactual: string }
  | { action: "outcome"; outcome: string; outcomeNote: string };

export const api = {
  meta: () => call<Meta>("/api/meta"),
  stats: () => call<Stats>("/api/stats"),

  me: () => call<User>("/api/me"),
  signup: (s: Signup) => call<User>("/api/auth/signup", json("POST", s)),
  login: (email: string, password: string) => call<User>("/api/auth/login", json("POST", { email, password })),
  logout: () => call<null>("/api/auth/logout", { method: "POST" }),

  dashboard: () => call<Dashboard>("/api/dashboard"),
  matches: (q: { material: string; kg: string; pieces: string; area: string }) =>
    call<Match[]>("/api/matches?" + new URLSearchParams(q)),
  createListing: (l: NewListing) => call<{ listing: Listing; waiting: number }>("/api/listings", json("POST", l)),
  withdrawListing: (id: string) => call<null>(`/api/listings/${encodeURIComponent(id)}/withdraw`, { method: "POST" }),
  draftFromPhoto: (photo: File) => {
    const form = new FormData();
    form.append("photo", photo);
    return call<Draft>("/api/listings/draft", { method: "POST", body: form });
  },

  needs: () => call<NeedView[]>("/api/needs"),
  saveNeed: (n: NewNeed) => call<SavedNeed>("/api/needs", json("POST", n)),
  closeNeed: (id: string) => call<null>(`/api/needs/${encodeURIComponent(id)}/close`, { method: "POST" }),

  transfers: () => call<Transfer[]>("/api/transfers"),
  request: (r: { listingId: string; intendedUse: string }) => call<Transfer>("/api/transfers", json("POST", r)),
  advance: (id: string, step: Step) => {
    const { action, ...body } = step;
    return call<Transfer>(
      `/api/transfers/${encodeURIComponent(id)}/${action}`,
      Object.keys(body).length ? json("POST", body) : { method: "POST" },
    );
  },
};
