// Package domain holds the rules of the materials exchange: what can be
// listed, where it may go, how a transfer moves through its lifecycle,
// how listings are matched to needs, and how much has been handed on.
package domain

// Material is a listable resource category.
type Material struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Examples string `json:"examples"`
}

var Materials = []Material{
	{"electronics", "Electronics and electrical", "Monitors, computers, circuit boards, cables"},
	{"plastics", "Plastics", "Clean containers, offcuts, packaging"},
	{"paper", "Paper and cardboard", "Boxes, packaging, paper offcuts"},
	{"metals", "Metals", "Metal offcuts, scrap metal"},
	{"wood", "Wood and timber", "Offcuts, planks, sawdust"},
	{"construction", "Construction materials", "Bricks, tiles, blocks"},
	{"textiles", "Textiles", "Fabric offcuts, unused fabric"},
	{"furniture", "Furniture and equipment", "Chairs, desks, shelves, tools"},
	{"agricultural", "Agricultural by-products", "Crop residues for composting"},
	{"other", "Other", "Anything else, reviewed before listing"},
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
