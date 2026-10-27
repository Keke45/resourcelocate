// Shapes of the JSON the Go API returns. Keep in sync with internal/domain.

// Catalog entries, served once by GET /api/meta.
export type Material = { id: string; label: string; examples: string };
export type Route = { id: string; label: string; description: string; recyclersOnly: boolean };
export type Outcome = { id: string; label: string };

export type Meta = {
  materials: Material[];
  routes: Route[];
  areas: string[];
  outcomes: Outcome[];
  roles: Role[];
  // How many reference photos a listing can carry.
  maxPhotos: number;
  // True when the server offers one-click demo logins for judges.
  demoLogins: boolean;
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
  // Counted things (monitors, desks) set pieces; bulk material sets estKg
  // (estimated total weight). Only one of the two is non-zero.
  pieces: number;
  estKg: number;
  condition: string;
  route: string;
  area: string;
  notes: string;
  status: ListingStatus;
  // Number of reference photos; see photoUrl.
  photos: number;
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

// How much changed hands: weight for bulk material, pieces for counted
// items, and the number of finished hand-overs.
export type Amount = { kg: number; pieces: number; handOvers: number };

// Totals count only finished hand-overs. given and claimed split them by
// side for the signed-in user; both are zero in community totals.
export type Totals = {
  handedOn: Amount;
  given: Amount;
  claimed: Amount;
  inProgress: number;
  byMaterial: (Amount & { material: string; label: string })[];
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
  totals: Totals;
};

export type Stats = { openListings: number; openNeeds: number; totals: Totals };
