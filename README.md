<div dir="rtl">

# خودروبین 🚗

**ترب برای خودروی دست‌دوم.** یک جمله‌ی فارسی معمولی بنویس — خودروبین آگهی‌ها را از سه منبع جمع می‌کند، عنوان‌های نامرتب فارسی را به یک ساختار واحد تبدیل می‌کند، آگهی‌های یک خودروی مشخص را زیر یک کارت جمع می‌کند، بر اساس نیت واقعی تو مرتب می‌کند، و با یک مدل زبانی می‌گوید **چرا** این گزینه بهترین است.

> ساخته‌شده برای چالش **AI Product Engineer** ترب: «ترب ___ رو بساز».

**دموی زنده:** <https://khodrobin.noxioai.com> · **ویدیوی ۵ دقیقه‌ای:** _(به‌زودی)_

### مسئله

یک پژو ۲۰۷ مدل ۱۴۰۴ هم‌زمان در دیوار، باما و همراه‌مکانیک آگهی می‌شود — با قیمت‌هایی از ۱٬۶۶۰٬۰۰۰٬۰۰۰ تا ۲٬۴۷۰٬۰۰۰٬۰۰۰ تومان. برخلاف کالای فروشگاهی، خودروی دست‌دوم **کد کالا ندارد**؛ هویتش باید از روی یک عنوان آزاد فارسی استنباط شود. این دقیقاً همان مسئله‌ی اصلی ترب است، در بازاری که سخت‌ترش می‌کند.

</div>

---

## English summary

**KhodroBin — "Torob for used cars."** Type one plain Persian sentence. KhodroBin collects listings from three Iranian marketplaces, normalizes messy free-text Persian into a canonical schema, groups every offer for the same car spec under one card, ranks by the user's real intent, and uses an LLM to explain *why* the top result wins.

Built for Torob's **AI Product Engineer** challenge. The five stages of their brief map onto the modules:

| Challenge line | Module | Status |
|---|---|---|
| `crawl offers` | [`crawler/sources/`](./crawler/sources) | ✅ 1,875 listings from 3 sources |
| `normalize messy data` | [`crawler/normalize.py`](./crawler/normalize.py) · [`plausibility.py`](./crawler/plausibility.py) | ✅ units, year systems, contradiction flags |
| `rank by user intent` | [`crawler/extract.py`](./crawler/extract.py) · `api/internal/rank` | 🚧 clustering done, ranking next |
| `explain the best choice` | `ai/` | 🚧 |
| `ship demo.mp4` | the video link above | 🚧 |

---

## What it does that a single listing site cannot

**One spec, every offer.** The unit of comparison is the car spec — brand, model, trim, gearbox, year, mileage band — with every matching offer underneath it and a real price distribution:

```
peugeot / 207 / base / manual / 1404      10 offers   {divar 2, hamrah 5, bama 3}
  median  2,114,000,000 تومان    range  1,660,000,000 .. 2,470,000,000
  hamrah  1,660,000,000  −21%     divar  1,920,000,000  −9%     bama  2,090,000,000  −1%
```

From 1,875 listings: **765 spec clusters, 69 of them spanning more than one source.**

**It tells you when a listing contradicts itself.** Bama lists a 1385 Pride as «صفر کیلومتر» while also recording «گلگیر تعویض» — a forty-year-old car with zero kilometres and body work. Divar uses `1,000,000 km` to mean "unknown" and returns placeholder prices as low as 10,000 rials. These are **flagged and shown, never silently repaired** — a repaired number is a lie with better manners.

**It reconciles units that do not agree.** Divar quotes rials; Bama and Hamrah-Mechanic quote tomans. Verified by comparing one car across sources: a 1385 Pride is `3,100,000,000` on Divar and `320,000,000` on Bama — the same ~315M tomans. All three also mix Jalali and Gregorian years *inside a single feed* depending on whether the car is domestic or imported.

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

Ranking is **deterministic Go** and unit-tested. The LLM does exactly three narrow jobs — parse intent, extract attributes the rules could not, and write the explanation — and every call is measured, cached, and evaluated against a golden set. Rules currently resolve **60%** of listings with no model calls at all.

### No browser automation

All three sources publish structured data over plain HTTP: Divar server-renders a schema.org `Car` array as JSON-LD, Bama has a public JSON search API, and Hamrah-Mechanic is Next.js with the payload in `__NEXT_DATA__`. `httpx` is enough.

Divar does push back — it stops including the JSON-LD once it decides you are crawling too fast, **without changing the HTTP status**. The crawler therefore treats an empty 200 as a soft block and backs off; see [`politeness.py`](./crawler/politeness.py).

---

## Running it

```bash
make up          # Postgres, Redis, API
make test        # Go race tests + Python tests
curl localhost:8080/healthz
```

```bash
cd crawler && python -m venv .venv && ./.venv/bin/pip install -r requirements.txt
./.venv/bin/python run.py --cities tehran mashhad isfahan --pages 30
```

A snapshot of 1,875 real listings ships in [`data/seed/`](./data/seed) so the demo works even when a source blocks us.

## Deployment

Everything ships from GitHub. Actions builds the image, publishes to GHCR, and the server only ever pulls — nothing is built or hand-edited on the box, so the repo is the single source of truth for what is running. Every push to `main` deploys and ends with a smoke test against the public URL.

Production is one Vultr box in Amsterdam (8 vCPU / 15 GB), chosen because it reaches **both** the model APIs and the Iranian sources — measured, not assumed. Caddy terminates TLS for `khodrobin.noxioai.com` and `khodro6.noxioai.com` on one certificate.

The submitted URL is never behind Vercel or Firebase: both are unreachable from Iran under US sanctions, and the reviewer for this application is in Tehran. See [`DECISIONS.md`](./DECISIONS.md) ۱۱.

## Docs

- [`DECISIONS.md`](./DECISIONS.md) — twelve decisions, each as context → options → choice → trade-off accepted
- [`docs/ROADMAP.md`](./docs/ROADMAP.md) — system design through scaling, the day-by-day plan, the video shot list

## License

MIT © Sobhan Azimzadeh
