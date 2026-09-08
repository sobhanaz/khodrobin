<div dir="rtl">

# خودروبین 🚗

**ترب برای خودروی دست‌دوم.** یک جمله‌ی فارسی معمولی بنویس. خودروبین آگهی‌ها را از پنج منبع جمع می‌کند، عنوان‌های نامرتب فارسی را به یک ساختار واحد تبدیل می‌کند، آگهی‌های یک خودرو را زیر یک کارت می‌آورد، بر اساس نیت واقعی تو مرتب می‌کند، و با یک مدل زبانی می‌گوید **چرا** این گزینه بهتر است.

> ساخته‌شده برای چالش **AI Product Engineer** ترب: «ترب ___ رو بساز».

**دموی زنده:** <https://khodrobin.noxioai.com>

### مسئله

یک ساینا مدل ۱۴۰۵ هم‌زمان در پنج سایت آگهی می‌شود، با قیمت‌هایی که چند برابر هم فرق دارند. برخلاف کالای فروشگاهی، خودروی دست‌دوم **کد کالا ندارد**؛ هویتش باید از روی یک عنوان آزاد فارسی استنباط شود. این دقیقاً همان مسئله‌ی اصلی ترب است، در بازاری که سخت‌ترش می‌کند.

</div>

---

## English summary

**KhodroBin — "Torob for used cars."** One plain Persian sentence in; a real market price out. It collects listings from five Iranian marketplaces, normalizes five incompatible schemas into one, groups every offer for the same car spec under one card, ranks deterministically by parsed intent, and uses an LLM to explain *why* the top result wins — with a guard that refuses to publish an explanation it cannot verify.

The five lines of Torob's brief map onto the modules:

| Challenge line | Module | Status |
|---|---|---|
| `crawl offers` | [`services/crawler/sources/`](./services/crawler/sources) | ✅ 5 sources, plain HTTP, every 3h |
| `normalize messy data` | [`normalize.py`](./services/crawler/normalize.py) · [`plausibility.py`](./services/crawler/plausibility.py) | ✅ units, year systems, contradiction flags |
| `rank by user intent` | [`services/api/internal/search`](./services/api/internal/search) | ✅ 4 modes, transparent score breakdown |
| `explain the best choice` | [`services/ai/`](./services/ai) | ✅ live, five-axis hallucination guard |
| `ship demo.mp4` | — | 🚧 the remaining piece |

---

# 1. System design

## The problem, stated precisely

Torob matches products that have a barcode. A used car does not have one. Two listings are the same car if a human reading both would say so, and nothing in the data says it directly.

The unit of comparison therefore has to be **invented**, and choosing it is the single most consequential decision in the project.

**The unit is the spec, not the physical car.**

```
brand · model · trim · gearbox · year · mileage-band
peugeot/207/panorama/mt/1405/0
```

Not the individual vehicle. Cross-source physical duplicates — the same VIN posted on two sites — do not exist at any honest volume, and pretending to detect them would mean inventing matches. Torob's own card is one *product* with many sellers, and a spec is the closest true analogue. Everything downstream follows from this: the median is a median over a spec, "cheapest" means cheapest for that spec, and a card is a spec.

## Four decisions that shaped everything

**Rules first, model second.** The deterministic parser resolves **56.1%** of listings and **90 of 92** golden queries with zero model calls. The model does exactly two narrow jobs: parsing queries the rules cannot, and writing the explanation. This is a cost and latency decision as much as a quality one — a search that costs a fraction of a cent per query cannot afford a model call on the hot path.

**Flag, never repair.** A contradiction is shown, not silently fixed. Bama lists a 1385 Pride as «صفر کیلومتر» while recording «گلگیر تعویض»; Divar uses `1,000,000 km` to mean "unknown". A repaired number is a lie with better manners. Flagged offers sink below clean ones and are withheld from the model.

**Expensive work happens offline.** Crawling, normalizing, clustering and explanation generation all run on a schedule. The request path only reads from an in-memory index. This is why search answers in about a millisecond and a precomputed explanation in under a second, on a 7B model that takes ten seconds to generate one.

**The guard is a guarantee; the prompt is only a request.** Every explanation is verified against its own input on five independent axes before it is allowed out.

## Where a model is genuinely needed, measured

The first golden set scored 100/100, and that was a warning rather than a win: it was written with knowledge of the parser, so it mostly asserted what was already implemented. A second set written from how people actually search scored **76.7%** with seven real failures. All were fixed in rules, because rules could genuinely do them.

A third set of misspellings scores **83.3%**, and its two remaining failures are the useful ones: they need Persian numbers written as words — «زیر **پونصد** میلیون», «مدل **نود و پنج**». That is where a regex engine stops being the right tool, and it is a **measured** argument for the model rather than an assumed one.

---

# 2. The dataset

## Shape, live

Read from [`/api/v1/stats`](https://khodrobin.noxioai.com/api/v1/stats):

```
captured        33,770   raw listings pulled from five sources
collapsed       18,007   the same ad seen again on a later crawl  (53%)
unique          15,763
indexed          8,837   listings whose identity could be resolved  (56.1%)
unresolved       6,926   kept, but not attributable to a spec
specs            4,584   the cards a user actually sees
multi-source       992   specs corroborated by more than one marketplace
flagged            380   offers that contradict themselves or their cohort
```

## The five sources, and how each betrays you

| Source | Transport | The trap |
|---|---|---|
| **Divar** | server-rendered `schema.org/Car` JSON-LD | quotes **rials**; soft-blocks with HTTP 200 and no data; `1,000,000 km` means "unknown"; instalment sales declared only in free-text title |
| **Bama** | public JSON search API | declares instalment sales in a **field**; mixes Jalali and Gregorian years in one feed |
| **Hamrah-Mechanic** | Next.js `__NEXT_DATA__` | — |
| **Khodro45** | Django REST endpoint | files **gearbox inside `trim`**, so a naive mapping keys every spec `gearbox=na` and builds a parallel universe of cards |
| **Sheypoor** | added last | verified against another source before trusting its unit |

**No browser automation.** All five publish structured data over plain HTTP; `httpx` is enough. Divar is readable precisely *because* it publishes JSON-LD for search engines — the same mechanism this project then uses for its own SEO.

## Three normalizations that decide whether any number is true

**Units.** Divar quotes rials; the other four quote tomans. Verified by comparing one real car across sources rather than trusting a field name: a 1385 Pride is `3,100,000,000` on Divar and `320,000,000` on Bama — the same ~315M tomans. Getting this wrong by 10× silently poisons every median in the index.

**Year systems.** All five mix Jalali and Gregorian **inside a single feed**, depending on whether the car is domestic or imported. `1405` and `2024` can appear in adjacent rows meaning the same model year.

**Deduplication.** 53% of captured rows are re-crawls of the same ad. Collapsed by `(source, source_id)` before anything counts them — an earlier content-hash approach covered fields that change on every fetch, so 43% of the index was one ad counted many times, skewing medians by up to 32%.

## Plausibility flags

```
zero_km_old_car        a 1385 car declared as having never been driven
zero_km_with_bodywork  «صفر کیلومتر» beside «گلگیر تعویض»
mileage_vs_age         implied km/year outside any plausible range
not_a_car_yet          «حواله» / «پیش‌فروش» — an allocation, not a car
instalment_price       «فروش اقساطی» — a down payment, not a price
price_outlier          far from the cohort median
no_price               no usable figure
```

Below three offers a median is the mean of two asking prices nobody is asking, so the card **says so** instead of quoting one.

---

# 3. System architecture

## Services

```
services/
  crawler/  Python   2,513 loc   5 sources → normalize → cluster → index → warm explanations
  api/      Go       2,343 loc   search, Persian query parsing, ranking, reverse lookup
  ai/       FastAPI  1,722 loc   intent, explanation, the guard, the eval harness
  auth/     Go       4,276 loc   accounts, sessions, saved searches, subscribers, contact
  web/      Nuxt 3   5,235 loc   SSR Persian RTL front end
```

## The two paths

**Offline, every 3 hours** — the expensive half:

```
crawl 5 sources  →  normalize units/years/schemas  →  flag contradictions
   →  collapse duplicates by (source, source_id)  →  cluster into specs
   →  write index.json  →  warm the top 150 explanations against the LLM
```

**Online, per request** — the cheap half:

```
query → Go parser (~0.25ms) → filter + rank in memory (~0.8ms) → JSON
                                     ↓ (only if the rules fail)
                              FastAPI → intent model
explain → precomputed file (~0.8s)  or  on-demand generation (~10s)
```

The API hot-reloads `index.json` via `atomic.Pointer` — a new crawl swaps the index with no restart and no dropped request.

## Request topology

```
                      ┌─────────── Caddy (TLS, one origin) ───────────┐
                      │                                               │
   /api/auth/*  →  auth:8081 ──┐                          /*  →  web:3000 (Nuxt SSR)
   /api/*       →  api:8080 ───┼── in-memory index ← indexdata volume ← crawler
   /healthz     →  api:8080    └── explain proxy   → ai:8000 → ollama (qwen2.5:7b)
                                                    auth → postgres, redis
```

Split at the edge rather than proxying auth through the search API: an outage in one must not take the other down, and **search stays anonymous**. Same origin throughout, so the browser makes no cross-origin request and there is no CORS surface at all.

## HTTP surface

**Search API** — `GET /api/v1/search` · `GET /api/v1/specs/{key}` · `GET /api/v1/stats` · `GET /api/v1/explain/{key}` · `POST /api/v1/lookup`

`POST /api/v1/lookup` is the reverse direction and the one nobody demos: paste a listing URL from any of the five marketplaces and get every offer for the same spec back, with an overpay verdict. The URL never leaves the process — it is parsed to source and id and answered from the index, so there is no fetch of the pasted page and no request surface a caller could point elsewhere. It is POST with the URL in a body because a listing URL in an access log or a referrer header is somebody's ad leaking into your logs.

**Accounts** — 24 routes under `/api/auth/*`. Seven tables: `users`, `sessions`, `tokens`, `saved_searches`, `alerts_sent`, `subscribers`, `contact_messages`.

## The guard

Five independent axes, because the model goes wrong in five different ways:

| Axis | Catches |
|---|---|
| numbers | a price or mileage never given |
| percentages | a position against the median no offer supports |
| topics | a subject the input never mentioned — warranty, paint, options |
| sources | an offer attributed to a marketplace with no listing here |
| **coherence** | a **true** statement that makes no sense |

Three exist because a real model produced the failure. Asked to explain a Peugeot 207, qwen2.5:7b invented a mechanical warranty — no number, so a numeric guard rated it clean. Percentages went unchecked entirely until an audit found the small-number filter was hiding the product's headline claim: «۴۵٪ زیر میانه» passed on a car 21.5٪ under.

The fifth is worth reading twice. A live card said «کارکرد صفر کیلومتر را **از دست می‌دهی**» — *in exchange, you lose the zero kilometres*. Every number in it is true, so all four factual axes passed it, and it is still nonsense. **10% of live explanations** framed the top offer's best feature as a sacrifice. Truth and sense are different properties, and only one of them was being checked.

When the guard fires, the card says so and shows what was rejected, then falls back to a sentence assembled from the data — always available, always true.

---

# 4. Performance

Measured against production, not a laptop.

| Path | Measured | How |
|---|---|---|
| Query parse | **0.22–0.28 ms** | Go, deterministic, no allocation on the hot path |
| Rank | **0.68–1.02 ms** | in-memory over 4,584 specs |
| Explanation, precomputed | **0.75–0.90 s** | 150 warmed offline every cycle |
| Explanation, cold | ~10 s | 7B model on CPU, no GPU |
| Model calls per search | **0** | rules resolve 90 of 92 golden queries |
| Cost per query | **$0.0000** | local model, no paid API on the request path |

Five consecutive live runs of the same query returned 407 results with parse times of 0.236 / 0.258 / 0.223 / 0.266 / 0.283 ms.

**Why the explanation is fast.** A 7B model on CPU takes about ten seconds. That is fine for a background job and unacceptable in a request, so the top 150 specs by corroboration are generated offline after each crawl and served from a file. The long tail still falls through to on-demand generation, so nothing is unreachable — it is just slower for a spec nobody has looked at yet.

**Cache invalidation is derived, not declared.** The warm file expires against a hash of the guard's own source, so tightening a rule expires every explanation the old rule approved, automatically. A version constant somebody has to remember to bump is documentation, not a mechanism — this one sat at `"1"` through a dozen guard changes.

## Query understanding

`make eval` grades the **live endpoint**, not a library. A parser unit test can pass while a stale index or a routing change has broken what users actually reach.

```
literal   50/50   100.0%     everyday queries
hard      30/30   100.0%     ranges, decimals, Finglish, brand-less models
messy     10/12    83.3%     misspellings and colloquial names
                             p50 1ms · $0.0000/query · 0 model calls
```

## Scaling posture

One Vultr box, Amsterdam, 8 vCPU / 15 GB — chosen because it reaches **both** the model APIs and the Iranian sources, measured rather than assumed. The index is 8,837 offers in memory; the same design holds to roughly a million before the in-process index needs to become a real store, and the seam is already there because the API reads through one interface.

---

## Frontend

**Nuxt 3 · Vue 3 · Tailwind v4**, server-rendered, Persian RTL throughout.

A Persian type scale rather than a Latin one wearing a Persian font. Two things every Latin system gets wrong here: **negative tracking** breaks Persian because the letters join, and **tight line height** crowds the marks Persian carries above and below the baseline. Vazirmatn is **self-hosted** — `fonts.gstatic.com` is unreliable from Iran and fails silently, dropping the page to Tahoma.

`/` is the landing page and `/search` is the application. `/search?q=` renders server-side, so a shared link arrives with results.

## For search engines

Every spec has a server-rendered page at `/car/{key}` publishing `schema.org/Car` markup — the same structured data Divar publishes, which is exactly what made them readable when this project crawled them. `sitemap.xml` is generated from the live index rather than stored, because the catalogue changes every three hours.

## Accounts

Search is anonymous and never sits behind a login. Accounts exist for **saved searches and price alerts**.

Security decisions worth stating: registration never reveals whether an address is registered, and login hashes a password even for accounts that do not exist so timing cannot become the same oracle; access tokens live in memory and refresh tokens in an HttpOnly cookie; refresh-token reuse is treated as compromise and kills every session; the admin route answers 404 rather than 403. **Marketing consent is honoured only at email verification**, because anyone can type anyone's address into a signup form — that is the entire reason verification exists.

---

## Running it

```bash
make up          # Postgres, Redis, API
make test        # Go race tests + Python tests
make index       # rebuild the search index from raw data
make eval        # grade query understanding against a running API
```

```bash
cd services/crawler && python -m venv .venv && ./.venv/bin/pip install -r requirements.txt
./.venv/bin/python run.py --cities tehran mashhad isfahan --pages 30
```

```bash
# serve the built front end against live data, for visual checks
cd services/web && NUXT_API_BASE=https://khodrobin.noxioai.com \
  NUXT_PUBLIC_API_BASE=https://khodrobin.noxioai.com PORT=3131 node .output/server/index.mjs
```

A seed snapshot ships in [`data/seed/`](./data/seed) so the demo works even when a source blocks us. In production the crawler re-runs every 3 hours and the API hot-reloads the new index without a restart.

**Tests:** 50 crawler · 33 AI guard · 4 Go `auth` packages (with `-race`) · 4 Go `api` packages. All gated in CI along with the eval thresholds and a Caddyfile validation.

## Deployment

Everything ships from GitHub. Actions builds every image, publishes to GHCR, and the server only pulls — nothing is built or hand-edited on the box, so the repo is the single source of truth for what is running.

The deploy gate probes what a visitor actually opens: readiness passes, the homepage renders, and a real query returns more than zero results. It used to curl `/healthz` alone, which asserted only that the process was running — something already true before the deploy.

The submitted URL is never behind Vercel or Firebase: both are unreachable from Iran under US sanctions, and the reviewer for this application is in Tehran.

## Docs

- [`DECISIONS.md`](./DECISIONS.md) — every decision as context → options → choice → trade-off accepted
- [`HANDOFF.md`](./HANDOFF.md) — current state, live numbers, what is next
- [`docs/ROADMAP.md`](./docs/ROADMAP.md) — system design through scaling, the video shot list

## License

MIT © Sobhan Azimzadeh
