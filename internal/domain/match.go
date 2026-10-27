package domain

import (
	"math"
	"sort"
)

// Need is what a seeker is looking for. The amount is either a weight (Kg)
// or a number of items (Pieces); at most one is set.
type Need struct {
	Material string  `json:"material"`
	Kg       float64 `json:"kg"`
	Pieces   int     `json:"pieces"`
	Area     string  `json:"area"`
	Role     string  `json:"role"`
}

type Match struct {
	Listing Listing  `json:"listing"`
	Score   int      `json:"score"`
	Reasons []string `json:"reasons"`
}

// Weights of the transparent match score; they add up to 100.
const (
	weightMaterial = 40
	weightQuantity = 25
	weightRoute    = 15
	weightArea     = 20
)

// Rank scores open listings against a need, best first. Listings the
// seeker may not request are left out. Ties go to the newest listing.
func Rank(listings []Listing, n Need) []Match {
	matches := []Match{}
	for _, l := range listings {
		if CanRequest(l, roleOr(n.Role)) != nil {
			continue
		}
		var score float64
		var reasons []string

		switch {
		case n.Material == "":
			score += weightMaterial
		case n.Material == l.Material:
			score += weightMaterial
			reasons = append(reasons, "material matches")
		default:
			continue
		}

		have, want := quantities(l, n)
		switch {
		case want <= 0:
			score += weightQuantity
		case have <= 0:
			// The listing is measured in the other unit (weight vs pieces):
			// we cannot compare, so give half marks rather than hide it.
			score += weightQuantity / 2
			if n.Pieces > 0 {
				reasons = append(reasons, "sold by weight")
			} else {
				reasons = append(reasons, "counted in pieces")
			}
		default:
			score += weightQuantity * math.Min(have/want, 1)
			if have >= want {
				reasons = append(reasons, "enough quantity")
			} else {
				reasons = append(reasons, "partial quantity")
			}
		}

		switch l.Route {
		case RouteReuse, RouteRepair:
			score += weightRoute
		case RouteRecycle, RouteSpecial:
			if n.Role == RoleRecycler {
				score += weightRoute
			} else {
				score += weightRoute / 3
				reasons = append(reasons, "needs processing")
			}
		}

		switch {
		case n.Area == "":
			score += weightArea
		case n.Area == l.Area:
			score += weightArea
			reasons = append(reasons, "same area")
		default:
			score += weightArea * 0.4
		}

		matches = append(matches, Match{Listing: l, Score: int(math.Round(score)), Reasons: reasons})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return matches[i].Listing.CreatedAt.After(matches[j].Listing.CreatedAt)
	})
	return matches
}

// quantities returns what the listing has and what the seeker wants, in
// the seeker's unit. have is 0 when the listing cannot be compared that way.
func quantities(l Listing, n Need) (have, want float64) {
	if n.Pieces > 0 {
		return float64(l.Pieces), float64(n.Pieces)
	}
	return l.EstKg, n.Kg
}

func roleOr(r string) string {
	if r == "" {
		return RoleIndividual
	}
	return r
}
