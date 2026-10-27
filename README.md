# ResourceLocate

ResourceLocate connects people who have unwanted materials with people who can use them, and shows how much has actually been given and claimed.

## The problem

Every day, households, businesses, workshops and institutions discard materials they no longer need. Some of these materials still have value. Others can be recycled or transformed into something useful.

At the same time, individuals, students, small businesses and community organisations need materials for projects, repairs, production and other activities.

The problem is that people who have unwanted materials often struggle to connect with people who could use them. So usable material is dumped, burned or landfilled while someone nearby goes without, or buys new.

## The idea

ResourceLocate is the missing connection. Anyone with unwanted material lists it in a minute, with photos if they like. Anyone who needs material finds it ranked by what, how much and how close. Recyclers see the items that need specialised handling.

A carpentry workshop's offcuts become a student's project. A school's old monitors go to a community computer room. Each hand-over is confirmed by both sides, so the totals show what really changed hands, not just what was listed.

Built for [Green Shift Hacks](https://green-shift-hacks.devpost.com/). It is piloted in Lusaka and built to work in any city.

## What it does

| Page | What happens |
| --- | --- |
| **Landing** | Public front page: the problem, what we believe (use before waste, connection is the missing piece, count only what really happened, safe by design, local first), how it works, who it's for, and live community totals. |
| **Sign up / Log in** | Email and password accounts. Each account has a name, a type (individual, business or group, recycler) and an area. One account can both give and receive. |
| **Dashboard** | What needs your attention, what you're waiting on, your saved needs with their current matches, your listings (with withdraw), and how much you've given and claimed. |
| **Find** | Search by material, amount (kg or pieces) and area. Results are ranked by a transparent score: material 40, quantity 25, route fit 15, same area 20. Save the search as a *need* and it stays on your dashboard, picking up new listings as they're posted. |
| **Give** | Describe the item, choose by weight (kg) or by count (pieces), and optionally add up to 4 reference photos so people looking can see what it is. Photos are shrunk in the browser before upload. You're told if people already have an open need for that material. |
| **Shared listing** | Every listing has its own link, `/l/{id}`, with a **Share** button on the dashboard, in the request window and on the page itself. On phones it opens the phone's share sheet (WhatsApp, SMS, Facebook, email and so on); elsewhere it copies the link. Anyone can open the link without an account. Chat apps show a preview with the item, amount, area and first photo. Visitors who aren't signed in can log in or sign up and land back on the listing to ask for it. The giver's phone number is never on this page. |
| **Settings** | Change your name and area, and change your password. Changing the password logs you out on every other device. |
| **Hand-overs** | Requested → accepted → collected → outcome recorded. The giver accepts and confirms the pickup (how many pieces, or how many kg). The receiver records what they did with it. The giver's phone number is shown to the receiver only after accepting. |

The app can be installed: Chrome, Edge and Samsung Internet show an **Install app** button (on phones, a download icon in the top bar); on an iPhone, use Share > Add to Home Screen. Installed or not, a service worker caches the app's pages, scripts, styles, fonts and icons, so it opens quickly on slow mobile data and still opens offline. Listings and hand-overs always come fresh from the server.

Every listing has one of four routes: direct reuse, repair or repurpose, recycling or recovery, or specialised handling. Specialised-handling items can only be requested by recyclers.

## What is counted

Simply what changed hands. A hand-over finishes when the giver confirms the pickup and the receiver records what they did with it. Only finished hand-overs count; a listing on its own doesn't.

- **Given:** what you handed over from your listings.
- **Claimed:** what you received from other people's listings.

Each listing is measured one way. Loose material such as offcuts, sawdust or fabric is listed by weight (kg). Separate items such as monitors, desks or chairs are listed by count (pieces). Totals keep the two apart, for example "45 kg and 12 pieces".

### Open records

`GET /api/records.csv` downloads every finished hand-over: date, material, item, area, receiver type, pieces or kg, and what the receiver did with it. It has no names, contacts or notes. The rows add up to the totals on the site, so anyone can check them.

## Run it

You need Go 1.26 or newer (Go 1.21 or later fetches the right version by itself on first run). Node 20 or newer is only needed if you change the front end.

```sh
go run ./cmd/server
# open http://localhost:8080
```

The built web app is committed in `internal/webui/dist` and embedded in the binary, so this one command runs everything. The app starts empty: create an account, then list something or save a need. Everything is stored in the SQLite file (`resource.db` by default) and is still there after a restart. To start over, stop the server and delete that file.

To try both sides of a hand-over, create two accounts, for example one in a normal window and one in a private window.

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

The tests cover password hashing, sessions and password changes, rate limiting, who may take each hand-over step, the transfer lifecycle, weight and counted listings, the recyclers-only rule, match ranking, withdrawing listings, saved needs, listing photos and the given and claimed totals.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port |
| `DB_PATH` | `resource.db` | SQLite file |
| `TRUST_PROXY` | unset | `1` when running behind a proxy that sets `X-Forwarded-For` (Render, Fly.io), so rate limits apply per visitor instead of to the proxy. Leave unset otherwise, since anyone can send that header. |
| `DEMO_LOGINS` | unset | `1` adds "Demo giver" and "Demo seeker" buttons on the login page, so judges can try both sides of a hand-over without signing up. The two accounts start empty and are shared by everyone who uses them. |

## Rate limits

Every API request is limited per IP address: bursts of 300, then 5 a second, photos included. Sign-up, login, demo login and password changes are limited much more tightly: 10 tries per IP, then one every 30 seconds. Logins are also limited per email address, 10 tries then one a minute, so guessing one account's password from many addresses is slow too. Over the limit the API answers `429` with a `Retry-After` header and the app shows how long to wait. The limits live in `internal/api/limits.go`; they are kept in memory, so they reset on restart and each server instance keeps its own.

## Deploy

The `Dockerfile` builds the web app and a static Go binary. Any host that runs a container works, such as Render, Fly.io or Railway. Mount a volume at `/data` to keep the database between deploys; without one, accounts and listings are lost on every deploy. Serve it over HTTPS: the session cookie is marked `Secure` when the request arrives over TLS or with `X-Forwarded-Proto: https`.

## Project layout

```
cmd/server         entry point: config and HTTP server
internal/domain    rules: catalog, accounts, lifecycle and permissions, matching, needs, totals (with tests)
internal/auth      password hashing (PBKDF2-SHA256) and session tokens
internal/store     SQLite storage for accounts, sessions, listings, needs and hand-overs
internal/ratelimit token-bucket rate limiter
internal/api       JSON API, rate limits
internal/webui     embedded React build
web/               React + TypeScript + Vite front end
web/public         app manifest, service worker (sw.js) and icons
```

## API

Public:

| Method and path | Purpose |
| --- | --- |
| `GET /api/meta` | Materials, routes, areas, roles and outcome options |
| `GET /api/stats` | Community totals for the landing page |
| `POST /api/auth/signup` | Create an account and log in |
| `POST /api/auth/login` | Log in (sets an HttpOnly session cookie) |
| `POST /api/auth/logout` | Log out |
| `POST /api/auth/demo` | `{"as":"giver"}` or `{"as":"seeker"}`: log in to a demo account (only with `DEMO_LOGINS=1`) |
| `GET /api/records.csv` | Every finished hand-over as CSV, without names |
| `GET /api/public/listings/{id}` | One listing for its share page (no phone number) |
| `GET /api/listings/{id}/photos/{n}` | Photo n of a listing (not served once the listing is withdrawn) |
| `GET /l/{id}` | The share page: the app with link-preview tags for that listing |

Signed in:

| Method and path | Purpose |
| --- | --- |
| `GET /api/me`, `PUT /api/me` | The signed-in account; change name and area |
| `POST /api/me/password` | `{"current": "...", "new": "..."}`: change password, logging out other devices |
| `GET /api/dashboard` | Your listings, needs with matches, hand-overs, and what you've given and claimed |
| `POST /api/listings` | Publish a listing |
| `POST /api/listings/{id}/withdraw` | Take down one of your open listings |
| `POST /api/listings/{id}/photos` | Add a reference photo (raw JPEG, PNG or WebP body, up to 2 MB, at most 4 per listing) |
| `GET /api/matches?material=&kg=&pieces=&area=` | Ranked open listings from other people |
| `GET /api/needs`, `POST /api/needs` | Your saved needs (with live matches), save a need |
| `POST /api/needs/{id}/close` | Remove a need |
| `GET /api/transfers` | Your hand-overs, as giver or receiver |
| `POST /api/transfers` | Request a listing |
| `POST /api/transfers/{id}/{accept,decline,collect,fail,outcome}` | Move a hand-over one step (giver or receiver only, as the step requires) |

## Limitations

- No email verification, and no "forgot password" email: a signed-in user can change their password in Settings, but someone who forgets it can't reset it themselves yet. Both need an email service.
- Amounts are what the giver confirms at hand-over; nothing weighs or counts them independently.
- Matching only knows whether two places are in the same area, not the distance between them.
- Listings can be withdrawn but not edited, and photos can't be changed after publishing.
