# Handoff — where KhodroBin stands

*Last updated: 8 Sep 2026. Target submission: 20 Sep 2026.*

> Every number in the **Live numbers** section below was read from the live API while writing
> this. Numbers elsewhere in this file predate that check unless they say otherwise.

---

## The one-paragraph state

**خودروبین is live at <https://khodrobin.noxioai.com>.** Five services on one box: a Python crawler over **five** Iranian marketplaces, a Go search API, a FastAPI model layer with a five-axis hallucination guard, a Go accounts service, and a server-rendered Nuxt front end in Persian RTL. Everything ships from GitHub — push to `main` → build → GHCR → deploy → smoke test. Four of the five rubric lines are implemented and deployed; the demo video is the one that remains.

---

## Latest session — 8 Sep 2026: email templates, OTP, and the agreed plan to win

> **Status corrected 8 Sep, later the same day.** This section described the email, OTP,
> Sheypoor, newsletter and landing work as uncommitted. It is now all committed and deployed
> (`0bddb1c` and earlier), and the working tree is clean. The five defects an adversarial review
> found in the subscribe flow are fixed; see **Bugs worth re-telling** items 12 and 13.

### What changed

| File | Change |
|---|---|
| `services/auth/internal/mail/templates.go` | Redesigned shared shell: preheader, `color-scheme: dark` (stops clients inverting the dark card), brand header chip, plain-text CTA fallback, labeled price panel. **`Verify` now takes `(link, code string)`.** Added `Welcome`, `PasswordChanged`, `SavedSearchCreated`, `codePanel`. |
| `services/auth/internal/mail/mail_test.go` | Updated for the new signature; new tests: preheader/fallback/color-scheme across all 7 mails, code panel, welcome, password-changed, saved-search. |
| `services/auth/internal/tokens/password.go` | New `Code()` — 6-digit OTP from crypto/rand with rejection sampling (no modulo bias; limit const `4_294_000_000`). |
| `services/auth/internal/handlers/handlers.go` | New route `POST /api/auth/verify/code` (+ `verifyCode`, `isSixDigits`, `verifyUser`, `sendWelcome`); `sendVerification` mints link **and** code tokens; reset sends a password-changed notice; arming an alert sends a confirmation mail. |
| `services/web/pages/verify.vue` | Code-entry form shown when no `?token=` is in the URL. |
| `email-preview/` (repo root) | 7 rendered HTML files for visual review. Disposable — delete when done. |

### How to verify

```bash
cd services/auth && gofmt -l internal/ && go test -race ./...
cd services/web && npm run build   # the CI gate; `nuxt typecheck` needs a generated tsconfig and is not in CI
```

Behavioral checks worth doing:
1. Register → the mail carries BOTH the link and a 6-digit code; either path verifies; the welcome mail arrives only on the first verification.
2. Reset → password-changed notice is sent; all sessions revoked (unchanged behavior).
3. Save a search **with an alert** → confirmation mail with query + threshold; a plain save sends nothing.
4. Wrong/expired code → «این کد معتبر نیست یا منقضی شده است.» — the same reply whether the code was used or never existed (no oracle).

### Gotchas

- `Verify(link, code)` is the only changed signature; only `sendVerification` calls it.
- Code TTL is 15 min (`codeTTL`), link TTL 24 h (`verifyTTL`) — the mail states both.
- Mail sends are fire-and-forget goroutines by design (the service never blocks on mail); they log `mail.sent` / warnings only.
- The whole tree is uncommitted (30+ files from earlier sessions + these). Nothing here has been through CI.

### Suggested next steps — agreed roadmap for the next agent

Prioritised with the owner (8 Sep). These are the improvements that make the
product bigger and more useful; each bullet names the outcome, not the steps.

**Tier 1 — finish what's half-done first**
1. Commit the whole working tree (Sheypoor source, newsletter, landing redesign, email work) — nothing has been through CI yet.
2. Record the demo video — the only unmet rubric line; shot list is ROADMAP.md §19.
3. Ops leaks: SPF/DKIM/DMARC so verification mail stops landing in spam; rotate the root password shared in chat on 6 Sep; grant the admin account (`UPDATE users SET is_admin = TRUE WHERE email = 'sobhandevuk@gmail.com'`).

**Tier 2 — product features (value ÷ effort, highest first)**
1. **Price-history chart per spec** — the data already exists (PriceAlert computes old/new medians); surface it as a 30/90-day trend on `/car/{key}` pages.
2. **Market report pages** — per-model pages (median price, active-offer count, cheapest city). SEO magnet, useful with no query.
3. **Dealer cards** — group listings by hashed seller phone: «این فروشنده ۴۲ آگهی دارد».
4. **Weekly digest email** — one template + a cron over saved searches; the email infra is done.
5. **Price distribution histogram** per spec — visual for the «زیر بازار» chip.
6. **PWA / push alerts** — manifest + service worker so price alerts reach the phone without email.

**Tier 3 — engineering depth (roadmap promises never built)**
1. Fill the empty packages: `api/internal/ratelimit/`, `cache/`, `metrics/` are `.gitkeep` only. Rate limiting (token bucket, `Retry-After`, `X-RateLimit-*`) and Redis result caching (`X-Cache: HIT`, SWR) are ROADMAP §§11–12.
2. Monitoring: `/metrics` + Prometheus + 6-panel dashboard, Telegram alerts, nightly eval cron (accuracy-regression alert).
3. Write `CRAWLING.md` and `SCALING.md` — required by the roadmap, both missing.
4. Postgres as source of truth with a proper ingest → index pipeline (API currently serves an embedded JSON index built by `build_index.py`).
5. Backups: nightly `pg_dump`, one tested restore (none exists).

**Tier 4 — explicitly not building (scoping discipline is part of the pitch)**
Accounts/payments/social login, native mobile app, image ML, national coverage, Elasticsearch at current scale.

### Wow features — proposed with the owner (8 Sep), not yet started

> These are the features picked for real "wow" — each one reuses machinery the
> repo already has. Outcome first, approach free.

**Data prerequisite for everything trend-based:** the index stores only the
*current* median per spec (and each offer's `seen_at`) — no history over time.
Add a daily snapshot `{spec_key, date, median_price, offer_count}` on each crawl
(the crawler already runs every 3 h). Without it, features 1–3 below are still
photos; with it, they unlock.

**Top 3 (recommended for the demo):**
1. **Car value estimator («قیمت ماشینت چنده؟»)** — attributes in → market range + where the listing sits. Torob's «قیمت محصول» pattern for cars; a lookup endpoint over existing spec data + the distribution math already computed.
2. **Cross-listing overpay alert** — when viewing any listing, «این ماشین ۳ جای دیگه هم آگهی شده — یکی ۱۸۰ میلیون ارزونتر». The dedupe already finds the same car across sources; repackage it as consumer protection. The 15-second demo.
3. **Momentum signal («الان بخرم یا صبر کنم؟»)** — per-spec verdict from the new price history: «داره میاد پایین، صبر کن» vs «داره میره بالا، وقتشه». Needs the snapshot.

**Also proposed (ranked):**
4. **Haggle-message generator** — LLM writes a polite Persian negotiation message grounded in verified market facts; the existing hallucination guard guarantees every number. Needs nothing new in the data layer.
5. **Telegram bot** — «قیمت ۲۰۶ تیپ ۵» in chat → spec card + alert. Reuses intent parser + search API.
6. **Persian voice search** — Web Speech API dictation into the same intent pipeline. Zero backend work, works in Chrome.
7. **Shareable price cards** — generated og-image per spec (gauge + median + trend) so links pasted in Telegram/WhatsApp show the price card.
8. **Market index ticker** — weighted index over top models' medians, «شاخص خودرو امروز: +۰.۸٪», 90-day line. Needs the snapshot.
9. **City price comparison** — same spec across cities («در مشهد ۳٪ ارزونتره»); city already exists on listings.
10. **Suspicious-listing feed** — public page of cars flagged for contradictions/outliers with the evidence shown; surfaces the honest flags already computed.

### Competition strategy — agreed with the owner (8 Sep): how to win among 300

> Context: a **300-person challenge** for Torob's AI Product Engineer role,
> submission 20 Sep. 295 competitors build the same competent thing. Winners
> separate on: (1) does it open from a phone in Tehran, (2) one unforgettable
> 15-second moment, (3) visible engineering depth. The project's moats are
> already the five-axis guard, the eval-gated CI, and 5-source identity
> resolution — **none of which 299 others have**.

**THE one idea to build first — «همین ماشین رو جای دیگه هم گذاشتن» (reverse lookup):**
Paste any listing URL from any source → KhodroBin finds the SAME car on the
other marketplaces → shows all prices → verdict: «این ماشین ۴ جای دیگه هم آگهی
شده — ارزونترینش ۱۸۰ میلیون ارزونتره». Why this one:
- It is Torob's stated hardest problem (same-item detection across stores)
  demonstrated in the direction nobody else will try (listing → same car
  everywhere, instead of search → results).
- It cannot be copied in 12 days: it requires the existing 5-source identity
  resolution + the `url` field already on every offer + `vs_median_pct` math.
- Build shape: `POST /api/v1/lookup` (extract source+id from URL → find the
  offer in the index → return spec, all offers, cheapest, overpay delta) + a
  paste-box on the homepage + a verdict card. ~1–2 days. This is the video's
  opening and closing shot.

**Second demo moment (if time):** the security lab — paste a hostile ad title
and watch the five-axis guard reject it live. **Third:** the haggle-message
generator (one call + the guard → copy-paste negotiation message).

### Decisions on record — 8 Sep (what the owner approved and passed on)

- ✅ **Approved & DONE this session:** email redesign (all 7 templates, one
  theme), OTP code flow (link AND 6-digit code, `POST /api/auth/verify/code`),
  welcome mail (first verification only), password-changed notice (every reset),
  alert-confirmation mail (when a threshold alert is armed).
- ✅ **Approved backlog:** the roadmap tiers above + all 10 wow features above.
- 🚫 **Passed on (deliberately NOT building):** full multi-agent LLM framework
  (the project already IS a two-agent system — intent + explain, each guarded;
  the only piece worth stealing is a **critic/verifier agent as a sampled
  spot-check**, never a gate), Elasticsearch at current scale, k8s/pods now,
  accounts/payments/social login, native mobile app, image ML, embeddings,
  feature flags, chaos testing. All belong in SCALING.md as write-don't-build.

### Docker / containers — the 7 fixes to reach perfect (audited 8 Sep)

The Docker layer is already strong (distroless nonroot, index baked in, Ollama
internal-only, healthchecks in dev, real smoke test). Remaining:
1. Prod compose: web/ai/auth/crawler/caddy have **no healthcheck or resource
   limits** — add both (roadmap §8 promised them on every container).
2. **Base images not pinned by digest** — `ollama:latest` is the risky one;
   pin it to a versioned tag and digest-pin the rest.
3. **Dockerfiles never built in CI** — breakage is caught at deploy time; add a
   CI job that builds all 5 images.
4. No container scans — add trivy (fail HIGH/CRITICAL), govulncheck, pip-audit.
5. **No backups at all** — nightly `pg_dump` to a second disk + off-box, one
   tested restore, screenshot the evidence. The only item that can end the
   submission if skipped.
6. No hardening caps — `read_only` rootfs + `cap_drop: [ALL]` on the two
   distroless services.
7. Deploy rolls forward, never back — smoke-test failure should pull the
   previous TAG and restart.

### Global-scale section map — verified status (8 Sep)

Full tickable list lives in **`docs/CHECKLIST.md`** (18 sections, status-keyed
✅/◐/⬜/🚫). The gaps that matter most, in order:
1. **Backups + tested restore** (can end the submission) — half day
2. **SPF/DKIM/DMARC** (verification mail lands in spam; blocks the accounts
   funnel) — 30 min of DNS
3. **CSP header** in Caddyfile + a `/privacy` page (you crawl personal data;
   Torob will look) — 1–2 h
4. **Inbound rate limiting + API `/metrics`** (the two empty `.gitkeep`
   packages — ghosts a reviewer WILL open) — 1–2 days
5. **Nightly eval cron + 2 Telegram alerts** (source-down, accuracy
   regression) — the video's best line
6. **k6 numbers + Dependabot + og:image per spec** (README evidence; links
   pasted in Telegram currently show a blank card)
7. Then the video, with the reverse-lookup moment as its spine.

Write-don't-build (SCALING.md only): multi-region, k8s, read replicas,
partitioning, tracing/OpenTelemetry, CDN, read-only rootfs is built not written.

### The order for the remaining 12 days (agreed)

1. **Commit the whole tree** (day 1) — nothing has been through CI.
2. **Build the reverse-lookup wow** (days 2–4) — the video needs the moment to
   exist before filming.
3. Rate limiting + `/metrics` (days 4–5).
4. Deliverability + admin + password rotation (day 5).
5. **Record/edit the video** around the reverse-lookup moment (days 6–10).
6. Docs (`CRAWLING.md`, `SCALING.md`), backups+restore evidence, nightly cron,
   Lighthouse screenshot (days 11–12).
7. **Test from an Irancell/MCI phone, Wi-Fi off** (day 12) — the check that
   decides more than everything else combined.

---

## Live numbers

Read from [`/api/v1/stats`](https://khodrobin.noxioai.com/api/v1/stats) on 8 Sep, not carried
forward from a previous version of this file:

```
captured 24,684  →  duplicates collapsed 11,094 (45%)  →  unique 13,590
indexed 7,591 offers  →  4,209 specs  →  842 multi-source  →  285 flagged
resolved by rules: 55.9%
sources: divar · bama · hamrah-mechanic · khodro45 · sheypoor
explanations warmed: 144
tests: 50 crawler · 33 AI guard · 4 Go auth packages · 3 Go api packages
```

Two of those moved sharply and both have a cause worth knowing. **Sheypoor** is now in the source
list, which is most of the jump in captured rows. **Flagged went from 26 to 285**, which is the
instalment-listing flag landing rather than a decline in data quality: «فروش اقساطی» quotes a down
payment, and 49 such listings had been leading their cards as the cheapest offer.

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
- **Explanations** from a local model, with a five-axis guard and a templated fallback that always passes it. Warmed ahead of time on a schedule: 149 precomputed, ~0.8s served instead of ~10s generated.
- **Accounts**: register → verify → login → saved searches → price alerts. SMTP confirmed working.
- **SEO**: per-car SSR pages with `schema.org/Car`, generated sitemap, generated robots.
- **CI**: gofmt, vet, race tests, Python tests, Nuxt build, Caddyfile validation, and the eval thresholds.

Pages: `/`, `/about`, `/faq`, `/contact`, `/login`, `/register`, `/verify`, `/account`, `/admin`, `/car/{key}`.

---

## Open items

**0. The price-alert checkbox on `/account` does nothing.** Verified rather than assumed:
`alerts_sent` is never written by any code, `last_median` appears in exactly one `SELECT` and is
never written, and no scheduler exists. The UI now says so instead of promising it. Wiring it up
is roughly half a day and completes the accounts story, which is the only justification the auth
service has: a ticker in the auth service re-runs alerting searches each crawl cycle, compares
medians, writes `alerts_sent`, and mails the difference. «پیگیری قیمت» is on Torob's own product,
so this is rubric surface, not a nice-to-have.

**0b. No Kubernetes exists.** Verified on the box: no kubectl, no k3s, no kubelet; `containerd` is
present only as Docker's runtime. Nine containers under Docker Compose on one host. Any monitoring
work should target Docker and the host, not pods.

**1. No admin account exists yet.** Registration creates ordinary users; `is_admin` defaults to false, deliberately — there is no self-promotion path in the code. To grant it after registering and verifying:

```sql
UPDATE users SET is_admin = TRUE WHERE email = 'sobhandevuk@gmail.com';
```

**2. SMTP deliverability.** Mail sends and is logged, but the domain has no SPF, DKIM or DMARC record, so verification email lands in spam — which is how the first admin account got stuck. Either add the three DNS records or move to a transactional provider; leaving it is a real signup funnel leak.

**3. The server root password is still the one shared in chat** on 6 Sep, and that transcript is not private. The Actions deploy key is installed and working, so rotating costs nothing operationally.

**4. Two audit workflows never finished.** Both were interrupted between sessions. Worth re-running now that auth, seven pages and the SEO surface exist — the attack surface has grown considerably since the first pass.

---

## Next, in priority order

> ⚠️ **Superseded 8 Sep** — the agreed plan above (commit → reverse-lookup wow →
> video → polish) replaces this section's ordering. Kept for the reasoning it
> contains; the video and the re-audit are still valid, but the reverse-lookup
> moment now comes first because the video needs it to exist.

### 1. The demo video — the only unmet rubric line
Shot list is in [`docs/ROADMAP.md`](./docs/ROADMAP.md) §19. Everything it needs to show now exists. The strongest sequence, in order: one car across five marketplaces → the messy Persian query parsed → `make eval` in a terminal → the guard rejecting a real model output live.

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
9. **The guard passed a sentence that was true and meaningless.** «کارکرد صفر کیلومتر را از دست می‌دهی» — *you lose the zero kilometres* — cleared all four factual axes because every number in it was correct. 10% of live explanations framed the top offer's best feature as a sacrifice.
10. **Tightening the guard changed nothing.** Both caches key on the data, so 149 warm explanations survived a rule change that would now reject 13 of them. `PROMPT_VERSION` had sat at `"1"` through a dozen prompt and guard edits — the safety mechanism was a constant someone had to remember, and nobody ever had.
11. **Instalment listings were priced as cars.** «فروش اقساطی» quotes a down payment, not the price of a car. 76 of them in the live index, and **49 were the cheapest offer on their card** — leading the result and being handed to the model as the choice to justify. The cause was an asymmetry, not a missing phrase: Bama declares instalment sales in a *field* and normalize.py has dropped them since the first crawl; Divar declares it only in free text and nothing looked. The check existed precisely where the data was already clean.
12. **A hover panel clipped a price to «۰,۰۰۰ تومان».** `position: fixed` is relative to the viewport only while no ancestor carries a transform, and `SpecCard` has `hover:-translate-y-0.5` — so hovering a source badge made the card the containing block at exactly the moment the panel opened. Correct coordinates, resolved against the wrong box. A layout bug was hiding a data bug: the instalment listing above had been leading cards for as long as Divar had been crawled, and nobody could see it because the panel that would have shown it rendered 588px off-screen.
13. **Registration confirmed addresses nobody owned.** The marketing opt-in was acted on immediately after the account row was written, before any verification click, so registering with a stranger's address and the box ticked put that address into the mailable export marked confirmed — and could confirm a pending double opt-in somebody else had genuinely started. **The hole had a passing test defending it**, asserting the vulnerable behaviour as the specification. Rewriting that test was the fix; making it pass would have been the bug.
14. **A password policy that was only a suggestion.** The browser checked ten runes, a symbol and a capital. The Go service checked length. Anything posting straight at the API got one rule out of three — beneath a comment claiming the server enforced all three independently.

**The recurring shape, now seen five times:** a comment asserting a check the code does not perform. The price-outlier flag that changed no behaviour, `PROMPT_VERSION = "1"`, the warm cache's carry-forward loop, `counts.mailable` reading a key the API has never emitted, and the password rules. None of them errored. All of them read as true.

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
