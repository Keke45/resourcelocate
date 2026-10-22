# ResourceLocate

ResourceLocate connects people who have unwanted materials with people who can use them, and counts only the kilograms that are actually reused or recycled.

## The problem

Every day, households, businesses, workshops and institutions discard materials they no longer need. Some of these materials still have value. Others can be recycled or transformed into something useful.

At the same time, individuals, students, small businesses and community organisations need materials for projects, repairs, production and other activities.

The problem is that people who have unwanted materials often struggle to connect with people who could use them. So usable material is dumped, burned or landfilled while someone nearby goes without, or buys new.

## The idea

ResourceLocate is the missing connection. Anyone with unwanted material lists it in a minute from a photo. Anyone who needs material finds it ranked by what, how much and how close. Recyclers see the items that need specialised handling.

A carpentry workshop's offcuts become a student's project. A school's old monitors go to a community computer room. Each hand-over is confirmed and its outcome recorded, so impact means verified waste diverted, not just listings.

Built for [Green Shift Hacks](https://green-shift-hacks.devpost.com/). It is piloted in Lusaka and built to work in any city.

## What it does

| Page | What happens |
| --- | --- |
| **Landing** | Public front page: the problem, what we believe (use before waste, connection is the missing piece, count only what really happened, safe by design, local first), how it works, who it's for, and live community totals. |
| **Sign up / Log in** | Email and password accounts. Each account has a name, a type (individual, business or group, recycler) and an area. One account can both give and receive. |
| **Dashboard** | What needs your attention, what you're waiting on, your saved needs with their current matches, your listings (with withdraw), and the impact of hand-overs you took part in. |
| **Find** | Search by material, amount (kg or pieces) and area. Results are ranked by a transparent score: material 40, quantity 25, route fit 15, same area 20. Save the search as a *need* and it stays on your dashboard, picking up new listings as they're posted. |
| **Give** | Take a photo and Claude drafts the listing (item, material, quantity, piece count, estimated kg, condition, route) and flags hazardous items. You check it and publish. You're told if people already have an open need for that material. |
| **Hand-overs** | Requested → accepted → collected → outcome recorded. The giver accepts and confirms the pickup (weight or pieces, plus "what would have happened otherwise?"). The receiver records what they did with it. The giver's phone number is shown to the receiver only after accepting. |

Every listing has one of four routes: direct reuse, repair or repurpose, recycling or recovery, or specialised handling. Specialised-handling items can only be requested by recyclers.

## How impact is counted

```
diverted kg   = handed-over kg × [would otherwise have been dumped, burned or landfilled]
CO₂e avoided  = diverted kg × emission factor for the material
```

A transfer counts only after the hand-over is confirmed and the outcome recorded. Material that would have been stored, sold or given away anyway counts as zero.

### Weight and pieces

Loose material such as offcuts, sawdust or fabric is listed by weight. Separate items such as monitors, desks or chairs are listed by count, with a rough weight per piece. When a school hands on 12 monitors, the app records 12 pieces and the dashboard shows them as items handed on.

Weight is still kept for every listing because diversion and CO₂e are calculated by weight: emission factors are per kilogram, and a monitor and a chair are not worth the same. At hand-over the receiver enters the number of pieces. A scale reading is optional for counted items. Without one, the weight comes from the provider's estimate per piece, and the dashboard shows how many of the kilograms were estimated this way.

> **The emission factors in `internal/domain/catalog.go` are placeholders.** Replace them with cited values, for example from the [US EPA WARM model](https://www.epa.gov/waste-reduction-model) or [UK government conversion factors](https://www.gov.uk/government/collections/government-conversion-factors-for-company-reporting), before quoting any CO₂e figure.

## Run it

You need Go 1.26 or newer (Go 1.21 or later fetches the right version by itself on first run). Node 20 or newer is only needed if you change the front end.

```sh
go run ./cmd/server
# open http://localhost:8080
```

The built web app is committed in `internal/webui/dist` and embedded in the binary, so this one command runs everything. The app starts empty: create an account, then list something or save a need. Everything is stored in the SQLite file (`resource.db` by default) and is still there after a restart. To start over, stop the server and delete that file.

To try both sides of a hand-over, create two accounts, for example one in a normal window and one in a private window.

To turn on photo-to-listing, set an Anthropic API key:

```sh
ANTHROPIC_API_KEY=sk-ant-... go run ./cmd/server
```

Without a key, the Give screen still works and every field is filled in by hand.

### Developing the front end

```sh
make dev-api   # terminal 1: API on :8080
make dev-web   # terminal 2: React on :5173, proxying /api to :8080
```

After changing anything in `web/`, run `cd web && npm run build` to refresh the embedded build.

### Tests

```sh
go test ./...
```

The tests cover password hashing, sessions, who may take each hand-over step, the transfer lifecycle, counted items, the recyclers-only rule, match ranking, withdrawing listings, saved needs and the impact calculation.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port |
| `DB_PATH` | `resource.db` | SQLite file |
| `ANTHROPIC_API_KEY` | unset | Turns on photo-to-listing |
| `CLAUDE_MODEL` | `claude-opus-5-5` | Model used to read photos |

## Deploy

The `Dockerfile` builds the web app and a static Go binary. Any host that runs a container works, such as Render, Fly.io or Railway. Mount a volume at `/data` to keep the database between deploys; without one, accounts and listings are lost on every deploy. Serve it over HTTPS: the session cookie is marked `Secure` when the request arrives over TLS or with `X-Forwarded-Proto: https`.

## Project layout

```
cmd/server         entry point: config and HTTP server
internal/domain    rules: catalog, accounts, lifecycle and permissions, matching, needs, impact (with tests)
internal/auth      password hashing (PBKDF2-SHA256) and session tokens
internal/store     SQLite storage for accounts, sessions, listings, needs and hand-overs
internal/vision    photo to draft listing with Claude
internal/api       JSON API
internal/webui     embedded React build
web/               React + TypeScript + Vite front end
```

## API

Public:

| Method and path | Purpose |
| --- | --- |
| `GET /api/meta` | Materials, routes, areas, roles, counterfactual and outcome options |
| `GET /api/stats` | Community totals for the landing page |
| `POST /api/auth/signup` | Create an account and log in |
| `POST /api/auth/login` | Log in (sets an HttpOnly session cookie) |
| `POST /api/auth/logout` | Log out |

Signed in:

| Method and path | Purpose |
| --- | --- |
| `GET /api/me` | The signed-in account |
| `GET /api/dashboard` | Your listings, needs with matches, hand-overs and impact |
| `POST /api/listings` | Publish a listing |
| `POST /api/listings/draft` | Multipart `photo` to draft listing |
| `POST /api/listings/{id}/withdraw` | Take down one of your open listings |
| `GET /api/matches?material=&kg=&pieces=&area=` | Ranked open listings from other people |
| `GET /api/needs`, `POST /api/needs` | Your saved needs (with live matches), save a need |
| `POST /api/needs/{id}/close` | Remove a need |
| `GET /api/transfers` | Your hand-overs, as giver or receiver |
| `POST /api/transfers` | Request a listing |
| `POST /api/transfers/{id}/{accept,decline,collect,fail,outcome}` | Move a hand-over one step (giver or receiver only, as the step requires) |

## Limitations

- No email verification or password reset yet, and no rate limit on login attempts. Add these before opening sign-up to the public.
- Weights read from photos are estimates. Only weights confirmed at hand-over count toward impact.
- The counterfactual is self-reported by givers.
- Matching only knows whether two places are in the same area, not the distance between them.
- Listings can be withdrawn but not edited yet.
