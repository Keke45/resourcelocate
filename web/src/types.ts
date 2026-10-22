// Shapes of the JSON the Go API returns. Keep in sync with internal/domain.

// Catalog entries, served once by GET /api/meta.
export type Material = { id: string; label: string; factorKgCO2e: number; examples: string };
export type Route = { id: string; label: string; description: string; recyclersOnly: boolean };
export type Counterfactual = { id: string; label: string; diverted: boolean };
export type Outcome = { id: string; label: string };

export type Meta = {
  materials: Material[];
  routes: Route[];
  areas: string[];
  counterfactuals: Counterfactual[];
  outcomes: Outcome[];
  roles: Role[];
  photoReading: boolean;
};

export type User = { id: string; name: string; email: string; role: Role; area: string; createdAt: string };

// open: anyone eligible can request it. reserved: a request was accepted.
// closed: the hand-over finished with a recorded outcome. withdrawn: the
// giver took it down.
export type ListingStatus = "open" | "reserved" | "closed" | "withdrawn";

export type Listing = {
  id: string;
  ownerId: string;
  providerName: string;
  contact: string;
  item: string;
  material: string;
  quantity: string;
  // Number of items for counted things (monitors, desks); 0 = by weight only.
  pieces: number;
  estKg: number;
  condition: string;
  route: string;
  area: string;
  notes: string;
  status: ListingStatus;
  createdAt: string;
};

export type Match = { listing: Listing; score: number; reasons: string[] | null };

export type Role = "individual" | "business" | "recycler";

// requested -> accepted -> collected -> completed. declined and failed are end
// states; a failed pickup reopens the listing.
export type TransferStatus = "requested" | "accepted" | "collected" | "completed" | "declined" | "failed";

export type Transfer = {
  id: string;
  listingId: string;
  receiverId: string;
  receiverName: string;
  receiverRole: Role;
  intendedUse: string;
  status: TransferStatus;
  confirmedKg: number;
  confirmedPieces: number;
  // True when confirmedKg comes from the per-piece estimate, not a scale.
  kgEstimated: boolean;
  counterfactual: string;
  outcome: string;
  outcomeNote: string;
  createdAt: string;
  updatedAt: string;
  item: string;
  material: string;
  route: string;
  providerName: string;
  area: string;
  listingPieces: number;
  listingEstKg: number;
  // Shared with the receiver once the request is accepted.
  contact: string;
  // Which side of the hand-over the signed-in user is on.
  side: "giving" | "receiving";
};

export type Draft = {
  item: string;
  material: string;
  quantity: string;
  pieces: number;
  estKg: number;
  condition: string;
  route: string;
  hazardous: boolean;
  hazardReason: string;
  confidence: "low" | "medium" | "high";
};

export type MaterialImpact = {
  material: string;
  label: string;
  handedKg: number;
  divertedKg: number;
  kgCO2e: number;
  pieces: number;
};

export type Impact = {
  completed: number;
  failed: number;
  inProgress: number;
  completionRate: number;
  handedKg: number;
  divertedKg: number;
  kgCO2e: number;
  pieces: number;
  estimatedKg: number;
  byMaterial: MaterialImpact[];
  byOutcome: Record<string, number>;
};

// A search saved on the user's account, with matches worked out fresh on
// every load so new listings show up.
export type SavedNeed = {
  id: string;
  material: string;
  kg: number;
  pieces: number;
  area: string;
  note: string;
  status: "open" | "closed";
  createdAt: string;
};
export type NeedView = SavedNeed & { matches: Match[] };

export type Dashboard = {
  listings: Listing[];
  needs: NeedView[];
  transfers: Transfer[];
  impact: Impact;
};

export type Stats = { openListings: number; openNeeds: number; impact: Impact };
