# Handoff — where KhodroBin stands

*Last updated: 7 Sep 2026. Target submission: 20 Sep 2026.*

---

## The one-paragraph state

**خودروبین is live at <https://khodrobin.noxioai.com>.** Five services on one box: a Python crawler over four Iranian marketplaces, a Go search API, a FastAPI model layer with a four-axis hallucination guard, a Go accounts service, and a server-rendered Nuxt front end in Persian RTL. Everything ships from GitHub — push to `main` → build → GHCR → deploy → smoke test. Four of the five rubric lines are implemented and deployed; the demo video is the one that remains.

---

## Live numbers

```
captured 10,516  →  duplicates collapsed 4,175 (40%)  →  unique 6,341
indexed 3,560 offers  →  2,286 specs  →  310 multi-source  →  26 flagged
sources: divar · bama · hamrah-mechanic · khodro45
search: parse ~0.01ms · rank ~1.4ms
eval: 90/92 golden queries, 0 model calls
tests: 106 across five services
```

Verify at [`/api/v1/stats`](https://khodrobin.noxioai.com/api/v1/stats) and `/readyz`.

---

## Infrastructure

| | |
|---|---|
| Server | Vultr Amsterdam, Ubuntu 26.04, 8 vCPU / 15 GB / 141 GB (address in GitHub secrets) |
| Domains | `khodrobin.noxioai.com` (A), `khodro6.noxioai.com` (AAAA), both DNS-only |
| Deploy path | `/opt/khodrobin`, `docker-compose.prod.yml` |
| Repo | <https://github.com/sobhanaz/khodrobin> (public) |
| Registry | GHCR — `khodrobin-{api,web,crawler,ai,auth}` |
| Model | Ollama `qwen2.5:7b`, pulled on boot, kept resident |

Containers: `caddy`, `web`, `api`, `ai`, `auth`, `crawler`, `ollama`, `postgres`, `redis`.

GitHub **secrets**: `DEPLOY_SSH_KEY`, `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_DOMAIN`, `POSTGRES_PASSWORD`, `SELLER_HASH_SALT`, `JWT_SECRET`, `SMTP_PASSWORD`.
GitHub **variables** (non-sensitive): `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_FROM`.

---

## What works

- **Four sources over plain HTTP.** No browser automation: Divar server-renders JSON-LD, Bama has a public JSON API, Hamrah-Mechanic is Next.js, Khodro45 is DRF.
- **Unit reconciliation, verified not assumed.** Divar quotes rials, the rest tomans; all four mix Jalali and Gregorian years inside one feed.
- **Deduplication.** 40% of captured rows are re-crawls of the same ad; collapsed before anything counts them.
- **Contradiction and outlier flags**, shown rather than silently repaired. Flagged offers sink below clean ones and never lead an explanation.
- **Deterministic ranking** in Go, four modes, returning its own per-factor breakdown — which is what makes the «پشت صحنه» panel honest.
- **Explanations** from a local model, with a four-axis guard and a templated fallback that always passes it.
- **Accounts**: register → verify → login → saved searches → price alerts. SMTP confirmed working.
- **SEO**: per-car SSR pages with `schema.org/Car`, generated sitemap, generated robots.
- **CI**: gofmt, vet, race tests, Python tests, Nuxt build, Caddyfile validation, and the eval thresholds.

Pages: `/`, `/about`, `/faq`, `/contact`, `/login`, `/register`, `/verify`, `/account`, `/admin`, `/car/{key}`.

---

## Open items

**1. No admin account exists yet.** Registration creates ordinary users; `is_admin` defaults to false, deliberately — there is no self-promotion path in the code. To grant it after registering and verifying:

```sql
UPDATE users SET is_admin = TRUE WHERE email = 'sobhandevuk@gmail.com';
```

**2. Explanation warming reports zero.** `/api/v1/stats` shows `explanations.warmed: 0`. On-demand explanations work (verified live), so this only costs first-visit latency on cold specs. Check `docker compose logs crawler` for the `warm_explanations.py` run and whether it writes `/data/explanations.json` before its budget expires.

**3. The server root password is still the one shared in chat** on 6 Sep, and that transcript is not private. The Actions deploy key is installed and working, so rotating costs nothing operationally.

**4. Two audit workflows never finished.** Both were interrupted between sessions. Worth re-running now that auth, seven pages and the SEO surface exist — the attack surface has grown considerably since the first pass.

---

## Next, in priority order

### 1. The demo video — the only unmet rubric line
Shot list is in [`docs/ROADMAP.md`](./docs/ROADMAP.md) §19. Everything it needs to show now exists. The strongest sequence, in order: one car across four marketplaces → the messy Persian query parsed → `make eval` in a terminal → the guard rejecting a real model output live.

### 2. Re-audit before recording
Adversarial passes over the guard, the live UX, the data fixes, and the eval methodology. Every previous round found something real that self-review had missed.

### 3. If time allows
Monitoring and alerting (Prometheus is exposed but nothing scrapes it), Elasticsearch (on Torob's stack list, currently unused), `CRAWLING.md` and `SCALING.md`.

---

## Bugs worth re-telling in the video

Each one passed every check while being broken. That is the point of the story: not that the code had bugs, but that the *tests and probes* said it did not.

1. **Divar soft-blocks with HTTP 200.** Under sustained crawling it stops including its JSON-LD and still returns 200. Success codes lie; the crawler measures yield.
2. **Caddy served the old front end for 49 minutes.** The Caddyfile is bind-mounted, so `compose up -d` never recreated it. Nuxt deployed, started healthy, and was never reached — and the smoke test passed, because `/healthz` still routed correctly.
3. **A fourth source changed nothing.** Khodro45 files the gearbox inside `trim`, so every spec keyed `gearbox=na` and silently formed a parallel universe. Fixed: 66 → 92 multi-source clusters at the time.
4. **43% of offers were the same ad counted again**, skewing medians by up to 32%, because the content hash covered fields that change on every fetch.
5. **The guard's most important check did not exist.** `IGNORE_BELOW=100` hid every percentage, so «۴۵٪ زیر میانه» passed on a car 21.5٪ under — and the test covering it passed only because the guard never looked.
6. **The guard then rejected correct output.** `11.6٪` parsed as `116` because `.` was treated as a thousands separator.
7. **Four fields silently dropped between Python and Go.** `encoding/json` discards unknown fields without error: car photos, `duplicates_collapsed`, `listings_captured`, `median_reliable`.
8. **A flagged outlier still led the card.** A «حواله» listing at 82% under its cohort was detected, tagged — and still shown first, and still handed to the model to justify.

---

## Useful commands

```bash
# local
make test                      # Go race tests + Python tests
make index                     # rebuild the index from raw data
make eval API=https://khodrobin.noxioai.com

cd services/api  && go test ./...
cd services/auth && go test ./...
cd services/crawler && ./.venv/bin/python -m pytest -q
cd services/ai   && ../crawler/.venv/bin/python -m pytest tests/ -q
cd services/web  && pnpm run build      # pnpm, not npm — see DECISIONS ۱۵

# server (host and credentials live in GitHub secrets)
ssh <deploy-user>@<deploy-host>
cd /opt/khodrobin
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs crawler --tail=30
docker compose -f docker-compose.prod.yml logs ai --tail=30
```

---

## Decisions on record

[`DECISIONS.md`](./DECISIONS.md) carries the full set. The load-bearing ones:

- **۱۰** — the split Iran/Frankfurt infrastructure was unnecessary; measured that Amsterdam reaches both the model APIs and the Iranian sources.
- **۱۱** — never Vercel or Firebase for the submitted URL; both are unreachable from Iran under sanctions and the reviewer is in Tehran.
- **۱۳** — the unit of comparison is the *spec*, not a physical car. Cross-source physical duplicates do not exist at any honest volume; Torob's own card is one product with many sellers.
- **۱۶** — accounts exist for saved searches and price alerts, and search never sits behind them.
