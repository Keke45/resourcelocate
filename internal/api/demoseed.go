package api

import (
	"context"
	"time"

	"github.com/KeKe45/resourcelocate/internal/auth"
	"github.com/KeKe45/resourcelocate/internal/domain"
)

// Demo activity: when demo logins are on, the exchange is filled with a few
// weeks of made-up but plausible activity around the two demo accounts, so
// judges see listings, saved needs, matches and hand-overs at every stage.
// The people and organisations are fictional, and so are their numbers.

// demoPerson is a fictional community member who lists, searches and
// collects alongside the demo accounts.
type demoPerson struct {
	key  string
	name string
	role string
	area string
}

var demoPeople = []demoPerson{
	{"tailors", "Matero Tailoring Co-op", domain.RoleBusiness, "Matero"},
	{"carpentry", "Kabwata Carpentry Works", domain.RoleBusiness, "Kabwata"},
	{"recycler", "GreenLoop Recyclers", domain.RoleRecycler, "Light Industrial Area"},
	{"printer", "Northmead Print House", domain.RoleBusiness, "Northmead"},
	{"student", "Mwila Banda", domain.RoleIndividual, "UNZA / Great East Road"},
	{"garden", "Chelstone Community Garden", domain.RoleBusiness, "Chelstone"},
	{"tiles", "Chanda Mulenga", domain.RoleIndividual, "Woodlands"},
	{"garage", "Kamwala Auto Workshop", domain.RoleBusiness, "Kamwala"},
}

const demoContact = "0970 000 000 (demo, not a real number)"

type demoListing struct {
	key, owner     string
	item, material string
	quantity       string
	pieces         int
	kg             float64
	condition      string
	route, area    string
	notes          string
	ago            time.Duration
	status         string
}

// Listing owners "giver" and "seeker" are the demo accounts; the rest are
// demoPeople keys. Status is set by the hand-overs below where they need it.
var demoListings = []demoListing{
	// The demo giver, a school clearing out storerooms.
	{key: "monitors", owner: "giver", item: "Old desktop monitors (17 inch)", material: "electronics", pieces: 12,
		condition: "Most switch on; a few have dead pixels", route: domain.RouteRepair, area: "Chilenje",
		notes: "From the old computer lab. Power cables included.", ago: 3 * day},
	{key: "exampaper", owner: "giver", item: "Used exercise books and exam paper", material: "paper", quantity: "6 sacks", kg: 80,
		condition: "Dry, sorted, no plastic covers", route: domain.RouteRecycle, area: "Chilenje", ago: 6 * day},
	{key: "chairs", owner: "giver", item: "Broken classroom chairs", material: "furniture", pieces: 30,
		condition: "Loose joints and missing backrests", route: domain.RouteRepair, area: "Chilenje", ago: 18 * day},
	{key: "boxes", owner: "giver", item: "Cardboard boxes from a book delivery", material: "paper", quantity: "About 40 boxes", kg: 25,
		condition: "Clean, flat-packed", route: domain.RouteReuse, area: "Chilenje", ago: 12 * day},
	{key: "curtains", owner: "giver", item: "Old classroom curtains", material: "textiles", quantity: "8 curtains", kg: 15,
		condition: "Faded but no tears", route: domain.RouteReuse, area: "Chilenje", ago: 2 * day},
	{key: "pallets", owner: "giver", item: "Wooden pallets", material: "wood", pieces: 10,
		condition: "Sturdy, a few cracked slats", route: domain.RouteReuse, area: "Chilenje",
		notes: "Left over from a furniture delivery.", ago: 1 * day},

	// The demo seeker, a maker who also passes things on.
	{key: "jars", owner: "seeker", item: "Glass jars, assorted sizes", material: "other", pieces: 40,
		condition: "Washed, lids for most", route: domain.RouteReuse, area: "Kabwata", ago: 4 * day},
	{key: "leather", owner: "seeker", item: "Leather offcuts", material: "textiles", quantity: "1 small bag", kg: 3,
		condition: "Pieces 5 to 20 cm", route: domain.RouteReuse, area: "Kabwata", ago: 20 * day},

	// The rest of the community.
	{key: "cotton", owner: "tailors", item: "Cotton fabric offcuts", material: "textiles", quantity: "3 sacks", kg: 25,
		condition: "Mixed colours, clean", route: domain.RouteReuse, area: "Matero", ago: 2 * day},
	{key: "denim", owner: "tailors", item: "Denim offcuts", material: "textiles", quantity: "2 sacks", kg: 18,
		condition: "Clean, mostly large pieces", route: domain.RouteReuse, area: "Matero", ago: 5 * day},
	{key: "chitenge", owner: "tailors", item: "Chitenge fabric offcuts", material: "textiles", quantity: "1 sack", kg: 12,
		condition: "Bright prints, 10 to 50 cm", route: domain.RouteReuse, area: "Matero", ago: 16 * day},
	{key: "desks", owner: "carpentry", item: "Refurbished student desks", material: "furniture", pieces: 6,
		condition: "Repaired and sanded", route: domain.RouteReuse, area: "Kabwata", ago: 2 * day},
	{key: "plywood", owner: "carpentry", item: "Plywood offcuts", material: "wood", quantity: "Half a pickup", kg: 40,
		condition: "12 mm and 18 mm, 20 to 80 cm", route: domain.RouteRepair, area: "Kabwata", ago: 7 * day},
	{key: "hardwood", owner: "carpentry", item: "Hardwood offcuts and sawdust", material: "wood", quantity: "5 sacks", kg: 120,
		condition: "Dry; sawdust bagged separately", route: domain.RouteReuse, area: "Kabwata",
		notes: "Good for small furniture, kindling or compost.", ago: 1 * day},
	{key: "timber", owner: "carpentry", item: "Pine timber offcuts", material: "wood", kg: 50,
		condition: "Short lengths", route: domain.RouteReuse, area: "Kabwata", ago: 24 * day},
	{key: "pet", owner: "recycler", item: "Sorted PET bottle flakes", material: "plastics", quantity: "3 bags", kg: 150,
		condition: "Washed and shredded", route: domain.RouteRecycle, area: "Light Industrial Area",
		notes: "For makers trying recycled plastic products.", ago: 3 * day},
	{key: "trimmings", owner: "printer", item: "Paper trimmings", material: "paper", quantity: "4 sacks", kg: 60,
		condition: "Clean white and coloured strips", route: domain.RouteRecycle, area: "Northmead", ago: 4 * day},
	{key: "plates", owner: "printer", item: "Aluminium printing plates", material: "metals", pieces: 20,
		condition: "Used, flat", route: domain.RouteRecycle, area: "Northmead", ago: 9 * day},
	{key: "laptops", owner: "student", item: "Broken laptops (for parts)", material: "electronics", pieces: 3,
		condition: "No batteries; screens work on two", route: domain.RouteRepair, area: "UNZA / Great East Road", ago: 1 * day},
	{key: "stalks", owner: "garden", item: "Maize stalks for compost", material: "agricultural", quantity: "A small truckload", kg: 300,
		condition: "Dry, cut", route: domain.RouteReuse, area: "Chelstone", ago: 6 * day},
	{key: "tiles", owner: "tiles", item: "Leftover floor tiles", material: "construction", pieces: 120,
		condition: "New, 40 x 40 cm, grey", route: domain.RouteReuse, area: "Woodlands",
		notes: "Left over after a renovation.", ago: 5 * day},
	{key: "bricks", owner: "tiles", item: "Bricks from a taken-down wall", material: "construction", pieces: 150,
		condition: "Cleaned of mortar", route: domain.RouteReuse, area: "Woodlands", ago: 14 * day},
	{key: "batteries", owner: "garage", item: "Used car batteries", material: "metals", pieces: 14,
		condition: "Dead, cases intact", route: domain.RouteSpecial, area: "Kamwala",
		notes: "Lead-acid: collection by a licensed recycler only.", ago: 2 * day},
	{key: "tyres", owner: "garage", item: "Worn car tyres", material: "other", pieces: 25,
		condition: "Too worn for the road", route: domain.RouteRepair, area: "Kamwala",
		notes: "Good for planters, swings or steps.", ago: 8 * day},
}

type demoNeed struct {
	owner, material string
	kg              float64
	pieces          int
	area, note      string
	ago             time.Duration
}

var demoNeeds = []demoNeed{
	{"seeker", "textiles", 20, 0, "Kabwata", "Fabric for tote bags and patchwork", 6 * day},
	{"seeker", "wood", 50, 0, "", "Timber for small furniture", 10 * day},
	{"seeker", "electronics", 0, 5, "", "Screens and laptops for a youth coding club", 3 * day},
	{"giver", "furniture", 0, 10, "Chilenje", "Desks for a new classroom", 8 * day},
	{"giver", "construction", 0, 200, "", "Bricks or tiles for a garden path", 4 * day},
	{"recycler", "paper", 500, 0, "", "Any clean paper or cardboard", 14 * day},
	{"recycler", "electronics", 0, 0, "", "E-waste for safe recycling", 14 * day},
	{"carpentry", "furniture", 0, 0, "", "Broken chairs and desks to repair", 11 * day},
	{"garden", "other", 0, 30, "", "Jars for seed storage", 5 * day},
	{"student", "electronics", 0, 2, "UNZA / Great East Road", "Monitors for a student coding club", 2 * day},
}

type demoTransfer struct {
	listing, receiver string
	use               string
	status            string
	kg                float64
	pieces            int
	outcome, note     string
	created, updated  time.Duration
}

var demoTransfers = []demoTransfer{
	// Waiting on the demo giver: two requests to answer and a pickup to confirm.
	{listing: "monitors", receiver: "recycler", use: "Test, repair what we can and recycle the rest",
		status: domain.TransferRequested, created: 1 * day, updated: 1 * day},
	{listing: "monitors", receiver: "student", use: "Two screens for a student coding club",
		status: domain.TransferRequested, created: 6 * hour, updated: 6 * hour},
	{listing: "exampaper", receiver: "recycler", use: "Pulp and recycle",
		status: domain.TransferAccepted, created: 4 * day, updated: 2 * day},
	{listing: "curtains", receiver: "seeker", use: "Tote bags and patchwork cushions",
		status: domain.TransferRequested, created: 5 * hour, updated: 5 * hour},
	// Finished hand-overs from the giver.
	{listing: "chairs", receiver: "carpentry", use: "Repair and return to classrooms",
		status: domain.TransferCompleted, pieces: 26, outcome: "repaired",
		note: "Fixed 26 chairs; the other 4 went for spare parts", created: 16 * day, updated: 12 * day},
	{listing: "boxes", receiver: "student", use: "Moving out of student housing",
		status: domain.TransferCompleted, kg: 22, outcome: "reused", note: "Used for moving, then passed on",
		created: 11 * day, updated: 9 * day},

	// The demo seeker's hand-overs as a receiver.
	{listing: "denim", receiver: "seeker", use: "Patchwork bags",
		status: domain.TransferAccepted, created: 3 * day, updated: 1 * day},
	{listing: "plywood", receiver: "seeker", use: "Shelves and small stools",
		status: domain.TransferCollected, kg: 35, created: 6 * day, updated: 2 * day},
	{listing: "plates", receiver: "seeker", use: "Wall art",
		status: domain.TransferDeclined, created: 8 * day, updated: 7 * day},
	{listing: "plates", receiver: "recycler", use: "Melt down and recycle",
		status: domain.TransferCompleted, pieces: 20, outcome: "recycled", created: 8 * day, updated: 6 * day},
	{listing: "chitenge", receiver: "seeker", use: "Tote bags for a market stall",
		status: domain.TransferCompleted, kg: 10, outcome: "repaired", note: "Made 14 tote bags",
		created: 15 * day, updated: 13 * day},
	{listing: "bricks", receiver: "giver", use: "A path through the school garden",
		status: domain.TransferCompleted, pieces: 150, outcome: "reused", note: "Laid a 20 m garden path",
		created: 13 * day, updated: 10 * day},
	// The demo seeker as a giver.
	{listing: "jars", receiver: "garden", use: "Seed storage for the garden",
		status: domain.TransferRequested, created: 1 * day, updated: 1 * day},
	{listing: "leather", receiver: "tailors", use: "Trims for bags",
		status: domain.TransferCompleted, kg: 3, outcome: "reused", created: 19 * day, updated: 15 * day},
	// Between other members.
	{listing: "timber", receiver: "tiles", use: "Raised garden beds",
		status: domain.TransferCompleted, kg: 45, outcome: "reused", note: "Built three raised beds",
		created: 22 * day, updated: 20 * day},
}

const (
	hour = time.Hour
	day  = 24 * time.Hour
)

// listingStatusAfter says what a listing's status is once its hand-overs
// have reached the given status.
var listingStatusAfter = map[string]string{
	domain.TransferAccepted:  domain.ListingReserved,
	domain.TransferCollected: domain.ListingReserved,
	domain.TransferCompleted: domain.ListingClosed,
}

// seedDemo fills the exchange with demo activity. It does nothing if the
// demo giver already has listings, so restarts don't add it twice and
// changes people make to the demo accounts are kept.
func (s *Server) seedDemo(ctx context.Context) error {
	giver := s.demo["giver"]
	if ls, err := s.store.ListingsBy(ctx, giver.ID); err != nil || len(ls) > 0 {
		return err
	}
	users := map[string]domain.User{"giver": giver, "seeker": s.demo["seeker"]}
	for _, p := range demoPeople {
		pw, _, err := auth.NewToken()
		if err != nil {
			return err
		}
		hash, err := auth.HashPassword(pw)
		if err != nil {
			return err
		}
		u := domain.User{Name: p.name, Email: "demo-" + p.key + "@resourcelocate.example", Role: p.role, Area: p.area}
		if err := s.store.EnsureUser(ctx, &u, hash); err != nil {
			return err
		}
		users[p.key] = u
	}

	now := time.Now().UTC()
	status := map[string]string{}
	for _, t := range demoTransfers {
		if next, ok := listingStatusAfter[t.status]; ok {
			status[t.listing] = next
		}
	}
	listings := map[string]domain.Listing{}
	for _, d := range demoListings {
		owner := users[d.owner]
		l := domain.Listing{
			OwnerID: owner.ID, ProviderName: owner.Name, Contact: demoContact, Item: d.item, Material: d.material,
			Quantity: d.quantity, Pieces: d.pieces, EstKg: d.kg, Condition: d.condition, Route: d.route,
			Area: d.area, Notes: d.notes, Status: status[d.key], CreatedAt: now.Add(-d.ago),
		}
		if err := domain.ValidateListing(&l); err != nil {
			return err
		}
		if err := s.store.CreateListing(ctx, &l); err != nil {
			return err
		}
		listings[d.key] = l
	}
	for _, d := range demoNeeds {
		n := domain.SavedNeed{UserID: users[d.owner].ID, Material: d.material, Kg: d.kg, Pieces: d.pieces,
			Area: d.area, Note: d.note, CreatedAt: now.Add(-d.ago)}
		if err := domain.ValidateNeed(&n); err != nil {
			return err
		}
		if err := s.store.CreateNeed(ctx, &n); err != nil {
			return err
		}
	}
	for _, d := range demoTransfers {
		r := users[d.receiver]
		t := domain.Transfer{
			ListingID: listings[d.listing].ID, ReceiverID: r.ID, ReceiverName: r.Name, ReceiverRole: r.Role,
			IntendedUse: d.use, Status: d.status, ConfirmedKg: d.kg, ConfirmedPieces: d.pieces,
			Outcome: d.outcome, OutcomeNote: d.note, CreatedAt: now.Add(-d.created), UpdatedAt: now.Add(-d.updated),
		}
		if err := s.store.InsertTransfer(ctx, &t); err != nil {
			return err
		}
	}
	return nil
}
