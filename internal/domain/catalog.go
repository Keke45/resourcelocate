// Package domain holds the rules of the materials exchange: what can be
// listed, where it may go, how a transfer moves through its lifecycle,
// how listings are matched to needs, and how impact is counted.
package domain

// Material is a listable resource category.
type Material struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// FactorKgCO2e is kg CO2e avoided per kg kept out of dumping or landfill.
	// These are placeholder values for the demo and must be replaced with
	// cited factors (for example US EPA WARM or UK conversion factors)
	// before any figure is presented as a result.
	FactorKgCO2e float64 `json:"factorKgCO2e"`
	Examples     string  `json:"examples"`
}

var Materials = []Material{
	{"electronics", "Electronics and electrical", 2.0, "Circuit boards, components, cables"},
	{"plastics", "Plastics", 1.5, "Clean containers, offcuts, packaging"},
	{"paper", "Paper and cardboard", 0.9, "Boxes, packaging, paper offcuts"},
	{"metals", "Metals", 2.0, "Metal offcuts, scrap metal"},
	{"wood", "Wood and timber", 0.5, "Offcuts, planks, sawdust"},
	{"construction", "Construction materials", 0.1, "Bricks, tiles, blocks"},
	{"textiles", "Textiles", 3.0, "Fabric offcuts, unused fabric"},
	{"furniture", "Furniture and equipment", 1.0, "Chairs, desks, shelves, tools"},
	{"agricultural", "Agricultural by-products", 0.2, "Crop residues for composting"},
	{"other", "Other", 0.5, "Anything else, reviewed before listing"},
}

// Route is where a provider intends the material to go.
type Route struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	// RecyclersOnly routes can only be requested by recyclers.
	RecyclersOnly bool `json:"recyclersOnly"`
}

const (
	RouteReuse   = "reuse"
	RouteRepair  = "repair"
	RouteRecycle = "recycle"
	RouteSpecial = "special"
)

var Routes = []Route{
	{RouteReuse, "Direct reuse", "Someone can use it as it is", false},
	{RouteRepair, "Repair or repurpose", "Needs fixing or can be made into something else", false},
	{RouteRecycle, "Recycling or recovery", "Needs processing by a recycler or recoverer", false},
	{RouteSpecial, "Specialised handling", "Only a qualified handler may collect it", true},
}

// Counterfactual answers "what would have happened to this otherwise?".
type Counterfactual struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Diverted means the hand-over kept the material out of dumping,
	// burning or landfill, so its weight counts toward impact.
	Diverted bool `json:"diverted"`
}

var Counterfactuals = []Counterfactual{
	{"dumped_burned", "Dumped or burned", true},
	{"landfill", "Sent to landfill", true},
	{"storage", "Kept in storage", false},
	{"given_anyway", "Given away or sold anyway", false},
}

type Outcome struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

var Outcomes = []Outcome{
	{"reused", "Reused directly"},
	{"repaired", "Repaired or repurposed"},
	{"recycled", "Recycled or recovered"},
	{"redirected", "Passed on to another appropriate destination"},
}

// Areas are Lusaka neighbourhoods offered in pickers. Matching only
// distinguishes same area from different area.
var Areas = []string{
	"Cairo Road (CBD)",
	"Chelstone",
	"Chilenje",
	"Emmasdale",
	"Kabulonga",
	"Kabwata",
	"Kamwala",
	"Light Industrial Area",
	"Matero",
	"Northmead",
	"UNZA / Great East Road",
	"Woodlands",
}

const (
	RoleIndividual = "individual"
	RoleBusiness   = "business"
	RoleRecycler   = "recycler"
)

var Roles = []string{RoleIndividual, RoleBusiness, RoleRecycler}

func MaterialByID(id string) (Material, bool) {
	for _, m := range Materials {
		if m.ID == id {
			return m, true
		}
	}
	return Material{}, false
}

func RouteByID(id string) (Route, bool) {
	for _, r := range Routes {
		if r.ID == id {
			return r, true
		}
	}
	return Route{}, false
}

func CounterfactualByID(id string) (Counterfactual, bool) {
	for _, c := range Counterfactuals {
		if c.ID == id {
			return c, true
		}
	}
	return Counterfactual{}, false
}

func ValidOutcome(id string) bool {
	for _, o := range Outcomes {
		if o.ID == id {
			return true
		}
	}
	return false
}

func ValidArea(a string) bool {
	for _, x := range Areas {
		if x == a {
			return true
		}
	}
	return false
}

func ValidRole(r string) bool {
	for _, x := range Roles {
		if x == r {
			return true
		}
	}
	return false
}
