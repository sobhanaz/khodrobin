<div dir="rtl">

# خودروبین 🚗

**ترب برای خودروی دست‌دوم.** یک جمله‌ی فارسی معمولی بنویس — خودروبین آگهی‌ها را از چهار منبع جمع می‌کند، عنوان‌های نامرتب فارسی را به یک ساختار واحد تبدیل می‌کند، آگهی‌های یک خودرو را زیر یک کارت می‌آورد، بر اساس نیت واقعی تو مرتب می‌کند، و با یک مدل زبانی می‌گوید **چرا** این گزینه بهتر است.

> ساخته‌شده برای چالش **AI Product Engineer** ترب: «ترب ___ رو بساز».

**دموی زنده:** <https://khodrobin.noxioai.com>

### مسئله

یک ساینا مدل ۱۴۰۵ هم‌زمان در چهار سایت آگهی می‌شود، با قیمت‌هایی که چند برابر هم فرق دارند. برخلاف کالای فروشگاهی، خودروی دست‌دوم **کد کالا ندارد**؛ هویتش باید از روی یک عنوان آزاد فارسی استنباط شود. این دقیقاً همان مسئله‌ی اصلی ترب است، در بازاری که سخت‌ترش می‌کند.

</div>

---

## English summary

**KhodroBin — "Torob for used cars."** One plain Persian sentence in; a real market price out. It collects listings from four Iranian marketplaces, normalizes four incompatible schemas into one, groups every offer for the same car spec under one card, ranks deterministically by parsed intent, and uses an LLM to explain *why* the top result wins — with a guard that refuses to publish an explanation it cannot verify.

The five lines of Torob's brief map onto the modules:

| Challenge line | Module | Status |
|---|---|---|
| `crawl offers` | [`services/crawler/sources/`](./services/crawler/sources) | ✅ 4 sources, plain HTTP, every 3h |
| `normalize messy data` | [`normalize.py`](./services/crawler/normalize.py) · [`plausibility.py`](./services/crawler/plausibility.py) | ✅ units, year systems, contradiction flags |
| `rank by user intent` | [`services/api/internal/search`](./services/api/internal/search) | ✅ 4 modes, transparent score breakdown |
| `explain the best choice` | [`services/ai/`](./services/ai) | ✅ live, five-axis hallucination guard |
| `ship demo.mp4` | — | 🚧 the remaining piece |

## Live numbers

```
captured 10,516  →  duplicates collapsed 4,175 (40%)  →  unique 6,341
indexed 3,560 offers  →  2,286 car specs  →  310 backed by more than one source
sources: divar · bama · hamrah-mechanic · khodro45
search: parse ~0.01ms · rank ~1.4ms
```

Read them live at [`/api/v1/stats`](https://khodrobin.noxioai.com/api/v1/stats).

---

## What it does that a single listing site cannot

**One spec, every offer.** The unit of comparison is the car spec — brand, model, trim, gearbox, year, mileage band — with every matching offer underneath it and a real price distribution. Each source badge links to the actual listing and previews it on hover.

**It reconciles units that do not agree.** Divar quotes rials; the other three quote tomans. Verified by comparing one car across sources rather than trusting field names: a 1385 Pride is `3,100,000,000` on Divar and `320,000,000` on Bama — the same ~315M tomans. All four also mix Jalali and Gregorian years *inside a single feed*, depending on whether the car is domestic or imported.

**It says when a listing contradicts itself.** Bama lists a 1385 Pride as «صفر کیلومتر» while recording «گلگیر تعویض». Divar uses `1,000,000 km` to mean "unknown". These are **flagged and shown, never silently repaired** — a repaired number is a lie with better manners.

**It refuses to quote a median it cannot stand behind.** Below three offers the median is the mean of two asking prices that nobody is asking, so the card says so instead.

---

## Measuring query understanding

`make eval` grades the **live endpoint**, not a library. A parser unit test can pass while a stale index or a routing change has broken what users actually reach.

```
$ make eval
literal   50/50   100.0%     everyday queries
hard      30/30   100.0%     ranges, decimals, Finglish, brand-less models
messy     10/12    83.3%     misspellings and colloquial names
                             p50 1ms · $0.0000/query · 0 model calls
```

**The first golden set scored 100/100, and that was a warning, not a win.** It was written with knowledge of the parser, so it mostly asserted what was already implemented. A second set written from how people actually search scored **76.7%** with seven real failures — year ranges, price ranges, decimal billions, bare Gregorian years, and models named without their brand. All were fixed in rules, because rules could genuinely do them.

A third set of misspellings then scored **75%**, and its two remaining failures are the useful ones: they need Persian numbers written as words — «زیر **پونصد** میلیون», «مدل **نود و پنج**». That is where a regex engine stops being the right tool, and it is a **measured** argument for the model rather than an assumed one.

---

## The guard

The interesting part of the AI layer is not the prompt. Every explanation is verified against its own input on five independent axes, because the model goes wrong in five different ways:

| Check | Catches |
|---|---|
| numbers | a price or mileage never given |
| percentages | a position against the median no offer supports |
| topics | a subject the input never mentioned — warranty, paint, options |
| sources | an offer attributed to a marketplace with no listing here |
| coherence | a **true** statement that makes no sense |

Three of these exist because a real model produced the failure. Asked to explain a Peugeot 207, qwen2.5:7b invented a mechanical warranty — no number, so a numeric guard rated it clean. Percentages went unchecked entirely until an audit found that the small-number filter was hiding the product's headline claim: «۴۵٪ زیر میانه» passed on a car that was 21.5٪ under.

The fifth is the one worth reading twice. A live card said «کارکرد صفر کیلومتر را **از دست می‌دهی**» — *in exchange, you lose the zero kilometres*. Every number in it is true, so all four factual axes passed it, and it is still nonsense. The cause is structural: the prompt asks the second sentence to name a trade-off, and when the cheapest offer is also the newest with the lowest mileage there is no trade-off to name. A model asked for one will supply one. Measured across the live index, **10% of explanations did this**. So the guard now rejects loss framing pointed at a dimension the top offer leads.

Truth and sense are different properties, and only one of them was being checked.

**A prompt is a request; a guard is a guarantee.** When the guard fires, the card says so and shows what was rejected, then falls back to a sentence assembled from the data — always available, always true. Verified across the whole index: every fallback satisfies the guard it exists to satisfy.

---

## Architecture

```
services/
  crawler/  Python   4 sources → normalize → cluster → index → explanation warming
  api/      Go       search, Persian query parsing, deterministic ranking, explain proxy
  ai/       FastAPI  intent, explanation, the guard, the eval harness
  auth/     Go       accounts, sessions, saved searches, price alerts, contact
  web/      Nuxt 3   SSR Persian RTL front end
```

| Layer | Choice | Why |
|---|---|---|
| Search core | **Go** | Ranking, caching, metrics; ~1 ms on the request path |
| Crawlers | **Python** | Plain HTTP — no browser automation needed |
| AI service | **Python / FastAPI** | Model clients, schema validation, **eval harness** |
| Accounts | **Go** + PostgreSQL | argon2id, rotating refresh tokens, email verification |
| Frontend | **Nuxt 3 · Vue 3 · Tailwind v4** | SSR, RTL-native, [`parsi-text`](https://github.com/sobhanaz/parsi-text) |
| Infra | Docker Compose · Caddy · GitHub Actions | One VPS, everything shipped from GitHub |

Ranking is **deterministic Go** and unit-tested; the model does exactly two narrow jobs and every call is measured, cached and evaluated. Rules resolve **56%** of listings and **90 of 92** golden queries with no model calls at all.

### No browser automation

All four sources publish structured data over plain HTTP: Divar server-renders a schema.org `Car` array as JSON-LD, Bama has a public JSON search API, Hamrah-Mechanic is Next.js with the payload in `__NEXT_DATA__`, and Khodro45 is a Django REST endpoint. `httpx` is enough.

Divar does push back — it stops including its JSON-LD once it decides you are crawling too fast, **without changing the HTTP status**. The crawler therefore measures *yield* and treats an empty 200 as a soft block; see [`politeness.py`](./services/crawler/politeness.py).

---

## For search engines

This is a search product, and a search product that search engines cannot read is competing with one hand tied. Every spec has a server-rendered page at `/car/{key}` publishing `schema.org/Car` markup — the same structured data Divar publishes, which is exactly what made them readable when this project crawled them.

`sitemap.xml` is generated from the live index rather than kept as a file, because the catalogue changes every three hours. Specs are ordered by corroboration: a crawl budget is finite, and a single-offer card is the least useful page here.

---

## Accounts

Search is anonymous and never sits behind a login. Accounts exist for **saved searches and price alerts** — Torob has «پیگیری قیمت», and a used-car price that moves is exactly the thing a buyer wants told to them.

Security decisions worth stating: the signup form never reveals whether an address is registered, and login hashes a password even for accounts that do not exist so timing cannot become the same oracle; access tokens live in memory and refresh tokens in an HttpOnly cookie, so an XSS bug borrows fifteen minutes rather than stealing thirty days; refresh-token reuse is treated as compromise and kills every session; the admin route answers 404 rather than 403; and arming a price alert requires a verified address, or the product becomes a way to mail strangers.

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

A seed snapshot ships in [`data/seed/`](./data/seed) so the demo works even when a source blocks us. In production the crawler re-runs every 3 hours and the API hot-reloads the new index without a restart.

**106 tests** across five services, all gated in CI along with the eval thresholds and a Caddyfile validation.

## Deployment

Everything ships from GitHub. Actions builds every image, publishes to GHCR, and the server only pulls — nothing is built or hand-edited on the box, so the repo is the single source of truth for what is running.

The deploy gate probes what a visitor actually opens: readiness passes, the homepage renders, and a real query returns more than zero results. It used to curl `/healthz` alone, which asserted only that the process was running — something already true before the deploy.

Production is one Vultr box in Amsterdam (8 vCPU / 15 GB), chosen because it reaches **both** the model APIs and the Iranian sources — measured, not assumed. The submitted URL is never behind Vercel or Firebase: both are unreachable from Iran under US sanctions, and the reviewer for this application is in Tehran.

## Docs

- [`DECISIONS.md`](./DECISIONS.md) — every decision as context → options → choice → trade-off accepted
- [`HANDOFF.md`](./HANDOFF.md) — current state, live numbers, what is next
- [`docs/ROADMAP.md`](./docs/ROADMAP.md) — system design through scaling, the video shot list

## License

MIT © Sobhan Azimzadeh
