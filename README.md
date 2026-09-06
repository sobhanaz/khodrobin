<div dir="rtl">

# خودروبین 🚗

**ترب برای خودروی دست‌دوم.** یک جمله‌ی فارسی معمولی بنویس — خودروبین آگهی‌ها را از چند منبع جمع می‌کند، عنوان‌های نامرتب فارسی را به یک ساختار واحد تبدیل می‌کند، تشخیص می‌دهد که سه آگهی در سه سایت **یک ماشین واقعی** هستند، بر اساس نیت واقعی تو مرتب می‌کند، و با یک مدل زبانی می‌گوید **چرا** این گزینه بهترین است.

> ساخته‌شده برای چالش **AI Product Engineer** ترب: «ترب ___ رو بساز».

**دموی زنده:** _(به‌زودی)_ · **ویدیوی ۵ دقیقه‌ای:** _(به‌زودی)_

### مسئله

یک پژو ۲۰۶ می‌تواند هم‌زمان در دیوار، باما و همراه‌مکانیک آگهی شود — با سه قیمت متفاوت، دو کارکرد متفاوت، و بدون هیچ راهی برای اینکه بفهمی این‌ها یک ماشین‌اند. برخلاف کالای فروشگاهی، خودروی دست‌دوم **کد کالا ندارد**؛ هویتش باید از روی یک عنوان آزاد فارسی استنباط شود. این دقیقاً همان مسئله‌ی اصلی ترب است، در بازاری که سخت‌ترش می‌کند.

</div>

---

## English summary

**KhodroBin — "Torob for used cars."** Type one plain Persian sentence. KhodroBin crawls listings from three Iranian marketplaces, normalizes messy free-text Persian titles into a canonical schema, resolves that three ads across three sites are **one physical car**, ranks by the user's real intent, and uses an LLM to explain *why* the top result wins.

Built for Torob's **AI Product Engineer** challenge. The five stages of their brief map 1:1 onto the modules:

| Challenge line | Module |
|---|---|
| `crawl offers` | [`crawler/internal/sources`](./crawler/internal/sources) |
| `normalize messy data` | [`crawler/internal/normalize`](./crawler/internal/normalize) · [`crawler/internal/identity`](./crawler/internal/identity) |
| `rank by user intent` | [`ai/app/intent`](./ai/app) · [`api/internal/rank`](./api/internal/rank) |
| `explain the best choice` | [`ai/app/explain`](./ai/app) |
| `ship demo.mp4` | the video link above |

### Why used cars

E-commerce products have model numbers. **Used cars have no SKU** — identity has to be inferred from a free-text Persian title, a price, a mileage and a year. That makes same-item detection, Torob's hardest listed problem, genuinely harder here, and it is the one place in this product where an LLM is *necessary* rather than decorative.

### The differentiator

One physical car, detected across three sites, collapsed into **one row with three prices** — plus a rolling market median so a listing can be flagged «۱۲٪ زیر بازار», a price-freshness flag, and a contradiction flag when two sources disagree about the same car.

---

## Stack

| Layer | Choice |
|---|---|
| Frontend | Nuxt 3 · Vue 3 · Tailwind · RTL/Persian-native · Vazirmatn · [`parsi-text`](https://github.com/sobhanaz/parsi-text) |
| Core API | **Go** — search, ranking, cache, rate limiting, metrics |
| Crawlers | **Go** — worker pool with per-host politeness limiting |
| AI service | **Python / FastAPI** — intent parsing, attribute extraction, explanation, **eval harness** |
| Data | PostgreSQL · Elasticsearch · Redis |
| Infra | Docker Compose · Caddy · GitHub Actions · Cloudflare |

Ranking is **deterministic Go** and unit-tested. The LLM does exactly three narrow jobs — parse intent, extract attributes the rules could not, and write the explanation — and every call is measured, cached, and evaluated against a golden set.

## Status

🚧 In active development. Day 0 of 14. See [`DECISIONS.md`](./DECISIONS.md) for the reasoning behind each choice.

## License

MIT © Sobhan Azimzadeh
