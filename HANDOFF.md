# Handoff — where KhodroBin stands

*Last updated: 7 Sep 2026, end of day 1 of 14. Target submission: 20 Sep 2026.*

---

## The one-paragraph state

**خودروبین is live at <https://khodrobin.noxioai.com>** (and `khodro6.noxioai.com` over IPv6). It crawls four Iranian car marketplaces on a 3-hour schedule, normalizes four incompatible schemas into one, groups offers by car spec, ranks them deterministically, and serves a server-rendered Persian RTL front end. Everything ships from GitHub: push to `main` → build → GHCR → deploy → smoke test. Nothing is built or edited on the server by hand.

---

## Live numbers (last measured)

```
listings 3,746    indexed 2,115 (56.5%)    specs 1,209    multi-source 137
sources: divar · bama · hamrah · khodro45
search: parse ~0.01ms · rank ~1.4ms
```

---

## Infrastructure

| | |
|---|---|
| Server | Vultr Amsterdam, Ubuntu 26.04, 8 vCPU / 15 GB / 141 GB (address held outside the repo) |
| Domains | `khodrobin.noxioai.com` (A), `khodro6.noxioai.com` (AAAA), both DNS-only |
| Deploy path | `/opt/khodrobin`, `docker-compose.prod.yml` |
| Repo | <https://github.com/sobhanaz/khodrobin> (public) |
| Registry | GHCR — `khodrobin-api`, `khodrobin-web`, `khodrobin-crawler` |

Services running: `caddy`, `web` (Nuxt), `api` (Go), `crawler` (Python), `postgres`, `redis`.

**Postgres and Redis are running but not yet used.** They are in the compose file for the next phase; the index is currently a JSON artifact on a shared volume.

GitHub secrets already set: `DEPLOY_SSH_KEY`, `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_DOMAIN`, `POSTGRES_PASSWORD`, `SELLER_HASH_SALT`.

---

## What works

- **Four sources, all over plain HTTP.** No browser automation: Divar server-renders JSON-LD `@type: Car`, Bama has a public JSON API, Hamrah is Next.js `__NEXT_DATA__`, Khodro45 is a DRF endpoint.
- **Unit reconciliation, verified not assumed.** Divar quotes rials; the other three quote tomans. All four mix Jalali and Gregorian years inside a single feed.
- **Contradiction flags.** e.g. Bama lists a 1385 Pride as «صفر کیلومتر» while recording «گلگیر تعویض». Flagged, never silently repaired.
- **Spec clustering** on brand/model/trim/gearbox/year/mileage-band, with median, price spread and per-source corroboration.
- **Deterministic ranking** in Go, 4 modes, returning its own per-factor breakdown (which the «پشت صحنه» panel draws).
- **Scheduled crawler** with a startup rebuild, atomic index writes, and hot-reload in the API — no restart needed.
- **Nuxt 3 + Vue 3 + Tailwind v4** front end, SSR, RTL-native.
- CI: gofmt, vet, race tests, Python tests, Caddyfile validation, Nuxt build.

Tests: 21 Python, ~17 Go.

---

## Open items — none blocking

Car photos are **live and verified**: the Go struct fix deployed and the API now
returns image URLs (confirmed from `cdn.hamrah-mechanic.com` and
`media.khodro45.com`). 96% of specs carry a photo.

Sanity check on resuming:

```bash
curl -s https://khodrobin.noxioai.com/healthz
curl -s "https://khodrobin.noxioai.com/api/v1/search?q=&limit=1" | grep -o '"image":"[^"]*"' | head -1
curl -s https://khodrobin.noxioai.com/api/v1/stats
```

Then open <https://khodrobin.noxioai.com> and confirm cards render with photos.

---

## Next, in priority order

### 1. The AI service (`services/ai/`) — not started
FastAPI with three narrow jobs and nothing else:
- `POST /intent` — Persian query → validated JSON. **The Go rules parser stays and always runs as the fallback**, so a model outage degrades search instead of breaking it.
- `POST /explain` — top 3 specs → two Persian sentences, constrained to facts in the input, with a post-validator that rejects any number not present in the input.
- Provider switch: `AI_PROVIDER=ollama|openai|anthropic|gemini`, one config line.

**Needs a decision:** an API key, or Ollama locally. The server reaches all three cloud providers (verified 401/405, not blocked) and has 15 GB RAM for Qwen 2.5 7B on CPU. Cloud is faster and gives a real cost number; Ollama costs nothing and always works.

### 2. `make eval` — golden set written, harness not
`services/ai/evals/golden_intents.jsonl` already has **50 hand-labelled Persian queries** covering: Finglish, Arabic-yeh folding, no-space input, two-digit Jalali years, price shorthand (بارsingle/میلیون/میلیارد), longest-alias cases (کرولا کراس vs کرولا), and the trap where «بالای ۹۵» is a year and not a price.

Still to write: `services/ai/evals/run_eval.py`. **Design decision already made:** it should hit the live `/api/v1/search` endpoint and compare the returned `intent`, so it measures the deployed system rather than a library in isolation. It can produce a real accuracy number for the rules parser *today*, before any model exists — that becomes the baseline the LLM must beat.

This is the highest-leverage remaining item: *"ساخت و بهتر کردن روش‌های ارزیابی دقت و عملکرد مدل‌های AI"* is a literal responsibility in the job description, and Torob publishes their own numbers (92% on 100 items, $6/day).

### 3. Then
- Monitoring: Prometheus `/metrics`, Grafana, Telegram alerts, nightly eval cron with regression alert.
- Elasticsearch (on their stack list, currently unused).
- `CRAWLING.md`, `SCALING.md`.
- The ≤4:50 Persian video. Shot list is in `docs/ROADMAP.md` §19.

---

## Bugs worth re-telling in the video

Each one passed every health check while being broken:

1. **Divar soft-blocks with HTTP 200.** Under sustained crawling it stops including its JSON-LD and still returns 200. Success codes lie; the crawler now measures *yield* and treats an empty 200 as a strike. Divar is throttled to 4s/request.
2. **Caddy served the old front end for 49 minutes.** The Caddyfile is bind-mounted, so editing it doesn't change the container spec and `compose up -d` won't recreate it. Nuxt deployed, started healthy, and was never reached — and the smoke test passed, because `/healthz` still routed correctly.
3. **Adding a fourth source changed nothing.** Khodro45 files the gearbox inside `trim`, so every spec keyed `gearbox=na` and silently formed a parallel universe. Fixed: 66 → 92 multi-source clusters, 2 → 14 with three sources.
4. **Photos vanished between Python and Go.** `encoding/json` drops unknown fields with no error.
5. **npm cannot resolve the Nuxt tree** — `Cannot read properties of null (reading 'edgesOut')`, reproducible with `nuxt + vue` alone on npm 10 and 11. Switched to pnpm.

---

## Decisions on record

`DECISIONS.md` has 15 entries, each as context → options → choice → trade-off. The load-bearing ones:

- **۱۰** — the split Iran/Frankfurt infrastructure was unnecessary; measured that Amsterdam reaches both the model APIs and the Iranian sources.
- **۱۱** — never Vercel or Firebase for the submitted URL; both are unreachable from Iran under sanctions and the reviewer is in Tehran.
- **۱۳** — the unit of comparison is the *spec*, not a physical car. Cross-source physical duplicates do not exist at any honest volume; Torob's own card is one product with many sellers, not one object.

---

## Useful commands

```bash
# local
cd services/api && go test ./...
cd services/crawler && ./.venv/bin/python -m pytest -q
cd services/web && pnpm run build

# rebuild the index locally from raw data
cd services/crawler && ./.venv/bin/python build_index.py \
  --raw ../../data/raw/listings.jsonl --out ../api/data/index.json

# server (password is in the session, not in this file)
ssh <deploy-user>@<deploy-host>
cd /opt/khodrobin && docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs crawler --tail=30
```

**Security note:** deployment credentials live only in GitHub Actions secrets, never in this repo. A dedicated Actions deploy key is installed on the box.
