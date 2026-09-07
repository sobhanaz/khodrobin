<div dir="rtl">

# خودروبین 🚗

**ترب برای خودروی دست‌دوم.** یک جمله‌ی فارسی معمولی بنویس — خودروبین آگهی‌ها را از چهار منبع جمع می‌کند، عنوان‌های نامرتب فارسی را به یک ساختار واحد تبدیل می‌کند، آگهی‌های یک خودروی مشخص را زیر یک کارت جمع می‌کند، بر اساس نیت واقعی تو مرتب می‌کند، و با یک مدل زبانی می‌گوید **چرا** این گزینه بهترین است.

> ساخته‌شده برای چالش **AI Product Engineer** ترب: «ترب ___ رو بساز».

**دموی زنده:** <https://khodrobin.noxioai.com> · **ویدیوی ۵ دقیقه‌ای:** _(به‌زودی)_

### مسئله

یک پژو ۲۰۷ مدل ۱۴۰۴ هم‌زمان در دیوار، باما، همراه‌مکانیک و خودرو۴۵ آگهی می‌شود — با قیمت‌هایی از ۱٬۶۶۰٬۰۰۰٬۰۰۰ تا ۲٬۴۷۰٬۰۰۰٬۰۰۰ تومان. برخلاف کالای فروشگاهی، خودروی دست‌دوم **کد کالا ندارد**؛ هویتش باید از روی یک عنوان آزاد فارسی استنباط شود. این دقیقاً همان مسئله‌ی اصلی ترب است، در بازاری که سخت‌ترش می‌کند.

</div>

---

## English summary

**KhodroBin — "Torob for used cars."** Type one plain Persian sentence. KhodroBin collects listings from four Iranian marketplaces, normalizes messy free-text Persian into a canonical schema, groups every offer for the same car spec under one card, ranks by the user's real intent, and uses an LLM to explain *why* the top result wins.

Built for Torob's **AI Product Engineer** challenge. The five stages of their brief map onto the modules:

| Challenge line | Module | Status |
|---|---|---|
| `crawl offers` | [`services/crawler/sources/`](./services/crawler/sources) | ✅ 7,600+ listings from 4 sources, every 3h |
| `normalize messy data` | [`services/crawler/normalize.py`](./services/crawler/normalize.py) · [`plausibility.py`](./services/crawler/plausibility.py) | ✅ units, year systems, contradiction flags |
| `rank by user intent` | [`services/api/internal/search`](./services/api/internal/search) | ✅ 4 modes, transparent score breakdown |
| `explain the best choice` | `services/ai/` | 🚧 next |
| `ship demo.mp4` | the video link above | 🚧 |

---

## What it does that a single listing site cannot

**One spec, every offer.** The unit of comparison is the car spec — brand, model, trim, gearbox, year, mileage band — with every matching offer underneath it and a real price distribution:

```
peugeot / 207 / base / manual / 1404      10 offers   {divar 2, hamrah 5, bama 3}
  median  2,114,000,000 تومان    range  1,660,000,000 .. 2,470,000,000
  hamrah  1,660,000,000  −21%     divar  1,920,000,000  −9%     bama  2,090,000,000  −1%
```

From 7,633 listings: **1,717 spec clusters, 194 of them spanning more than one source.**

**It tells you when a listing contradicts itself.** Bama lists a 1385 Pride as «صفر کیلومتر» while also recording «گلگیر تعویض» — a forty-year-old car with zero kilometres and body work. Divar uses `1,000,000 km` to mean "unknown" and returns placeholder prices as low as 10,000 rials. These are **flagged and shown, never silently repaired** — a repaired number is a lie with better manners.

**It reconciles units that do not agree.** Divar quotes rials; the other three quote tomans. Verified by comparing one car across sources: a 1385 Pride is `3,100,000,000` on Divar and `320,000,000` on Bama — the same ~315M tomans. All four also mix Jalali and Gregorian years *inside a single feed* depending on whether the car is domestic or imported.

---

## Measuring query understanding

`make eval` grades the **live endpoint**, not a library. A parser unit test can
pass while a stale index or a routing change has broken what users actually
reach, so the harness sends each golden query to a running `/api/v1/search` and
grades the `intent` that comes back.

```
$ make eval
literal   50/50   100.0%     everyday queries
hard      30/30   100.0%     ranges, decimals, Finglish, brand-less models
messy     10/12    83.3%     misspellings and colloquial names
                             p50 1ms · $0.0000/query · 0 model calls
```

**The first golden set scored 100/100, and that was a warning, not a win.** It
was written with knowledge of the parser, so it mostly asserted what was already
implemented. A second set written from how people actually search scored
**76.7%** with seven real failures — year ranges, price ranges, decimal
billions, bare Gregorian years, and models named without their brand. All were
fixed in rules, because rules could genuinely do them.

A third set of misspellings and colloquial names then scored **75%**, and its
remaining two failures are the useful ones: they need Persian numbers written as
words — «ماشین زیر **پونصد** میلیون», «مدل **نود و پنج**». That is where a regex
engine stops being the right tool, and it is a **measured** argument for adding a
model rather than an assumed one.

Thresholds differ per set on purpose. The solved sets gate at 100% so any drop
is a regression; the messy set gates at its current 83% as a floor that rises
when the model lands. Both run in CI on every push.

---

## Stack

| Layer | Choice | Why |
|---|---|---|
| Core API | **Go** | Search, ranking, cache, rate limiting, metrics |
| Crawlers | **Python** | Plain HTTP — no browser automation needed (see below) |
| AI service | **Python / FastAPI** | Intent, attribute extraction, explanation, **eval harness** |
| Frontend | **Nuxt 3 · Vue 3 · Tailwind** | RTL/Persian-native, [`parsi-text`](https://github.com/sobhanaz/parsi-text) |
| Data | PostgreSQL · Elasticsearch · Redis | Relational aggregation is the product |
| Infra | Docker Compose · Caddy · GitHub Actions | One VPS, everything shipped from GitHub |

Ranking is **deterministic Go** and unit-tested. The LLM does exactly three narrow jobs — parse intent, extract attributes the rules could not, and write the explanation — and every call is measured, cached, and evaluated against a golden set. Rules currently resolve **54%** of listings, and **90 of 92** golden queries, with no model calls at all.

### No browser automation

All four sources publish structured data over plain HTTP: Divar server-renders a schema.org `Car` array as JSON-LD, Bama has a public JSON search API, Hamrah-Mechanic is Next.js with the payload in `__NEXT_DATA__`, and Khodro45 is a Django REST endpoint. `httpx` is enough.

Divar does push back — it stops including the JSON-LD once it decides you are crawling too fast, **without changing the HTTP status**. The crawler therefore treats an empty 200 as a soft block and backs off; see [`politeness.py`](./services/crawler/politeness.py).

---

## Running it

```bash
make up          # Postgres, Redis, API
make test        # Go race tests + Python tests
curl localhost:8080/healthz
```

```bash
make eval        # grade query understanding against a running API
make index       # rebuild the search index from raw data

cd services/crawler && python -m venv .venv && ./.venv/bin/pip install -r requirements.txt
./.venv/bin/python run.py --cities tehran mashhad isfahan --pages 30
```

A seed snapshot ships in [`data/seed/`](./data/seed) so the demo works even when a source blocks us. In production the crawler re-runs every 3 hours and the API hot-reloads the new index without a restart.

## Deployment

Everything ships from GitHub. Actions builds the image, publishes to GHCR, and the server only ever pulls — nothing is built or hand-edited on the box, so the repo is the single source of truth for what is running. Every push to `main` deploys and ends with a smoke test against the public URL.

Production is one Vultr box in Amsterdam (8 vCPU / 15 GB), chosen because it reaches **both** the model APIs and the Iranian sources — measured, not assumed. Caddy terminates TLS for `khodrobin.noxioai.com` and `khodro6.noxioai.com` on one certificate.

The submitted URL is never behind Vercel or Firebase: both are unreachable from Iran under US sanctions, and the reviewer for this application is in Tehran. See [`DECISIONS.md`](./DECISIONS.md) ۱۱.

## Docs

- [`DECISIONS.md`](./DECISIONS.md) — fifteen decisions, each as context → options → choice → trade-off accepted
- [`HANDOFF.md`](./HANDOFF.md) — current state, live numbers, what is next
- [`docs/ROADMAP.md`](./docs/ROADMAP.md) — system design through scaling, the day-by-day plan, the video shot list

## License

MIT © Sobhan Azimzadeh
