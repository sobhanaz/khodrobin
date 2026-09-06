<div dir="rtl">

# تصمیم‌ها

هر تصمیم به این شکل ثبت می‌شود: **زمینه → گزینه‌ها → انتخاب → هزینه‌ای که پذیرفتیم.**

</div>

Each decision is recorded as **context → options → choice → trade-off accepted.** Written as the project is built, not afterwards.

---

## ۱. چرا خودروی دست‌دوم، نه بلیت هواپیما — Why used cars

**Context.** Torob's brief names four example markets: بلیت هواپیما، بیمه، خودرو، خانه. Their hardest listed engineering problem is «تشخیص یکسان بودن کالاها در فروشگاه‌های مختلف» — deciding that a listing on store A and a listing on store B are the same thing.

**Options.**
- *Flights* — Torob's own first example. Natural keys exist (airline + flight number + departure), so identity resolution is mostly a join. Safe, obvious, and the market most candidates will pick.
- *Insurance* — pricing requires authenticated quote flows; hard to crawl honestly.
- *Housing* — big duplicate problem, but attributes are mushy and price comparison is weak.
- *Used cars* — no SKU. Identity must be inferred from a free-text Persian title.

**Choice.** Used cars.

**Why.** Flights hand you the natural key; cars make you earn it. «پژو ۲۰۶ تیپ۵ مدل۹۰ فول» has to become `{brand, model, trim, year}` before two ads can even be compared. That is the one place in this product where an LLM is genuinely necessary rather than decorative — and demonstrating that distinction is what the role is actually testing.

**Trade-off accepted.** Identity resolution is now a real problem with a real error rate instead of a join, so it needs its own confidence model, its own golden set, and a visible way for the user to disagree with it.

---

## ۲. چرا هم Go و هم Python — Why Go *and* Python

**Context.** The job description's first hard expectation is «تجربه کاری قوی با Python و فریم‌ورک Django». My depth is Go, Vue/Nuxt and infrastructure. The challenge page imposes no stack.

**Options.** All Go (fastest for me, ignores their first line) · all Python (slower for me, abandons my strength) · split by responsibility.

**Choice.** Go for the core API, ranking, caching and crawlers. Python/FastAPI for a small, focused AI service: intent parsing, attribute extraction, explanation, and the evaluation harness. They talk over a small HTTP contract with a strict schema.

**Why.** This is not a hedge — it is where each language is actually better. The search path fans out to several indexes and must hold p95 under 300 ms, which is Go's job. The evaluation tooling, model clients and schema validation ecosystem live in Python, and the eval harness belongs next to the model calls it measures.

**Trade-off accepted.** Two runtimes, two test suites, one more container, and a network hop on the request path — mitigated by caching the intent and explanation results.

---

## ۳. مدل فقط سه کار باریک انجام می‌دهد — The LLM does exactly three narrow jobs

**Context.** The easy version of this product is a chatbot over car listings. It demos well for thirty seconds and is unshippable.

**Choice.** The model does three things only: parse a Persian query into validated JSON, extract attributes from titles the rules could not confidently parse, and write a two-sentence explanation constrained to facts already in its input. Retrieval, deduplication scoring and ranking are deterministic code with unit tests.

**Why.** Ranking must be explainable, testable and cheap. A weighted score I can unit-test and draw as a bar chart beats a model I have to trust. It also means a model outage degrades the product instead of breaking it — the keyword fallback parser still answers.

**Trade-off accepted.** Less "magic" in the demo. Compensated by making the machinery visible: every result opens to show its sources, its normalization decisions, its score breakdown and the exact prompt.

---

## ۴. قانون اول، مدل دوم — Rules first, model second

**Context.** Running an LLM over every crawled title is simple, slow and expensive.

**Choice.** The normalizer tries regex, a dictionary of Iranian car models and trims, and Persian text normalization first. Only rows below a confidence threshold go to the model. The split is logged and reported.

**Why.** Most Persian car titles are formulaic. Paying a model to parse «پژو ۲۰۶ تیپ ۵ مدل ۱۳۹۶» is waste. Reporting "83% by rules, 17% by model" is a cost-discipline number, and cost discipline is something Torob publishes about their own systems.

**Trade-off accepted.** A dictionary to maintain, and a threshold to tune against the golden set.

---

## ۵. داده‌ی seed در مخزن می‌ماند — Seed data ships in the repo

**Context.** A reviewer opens the live demo at an unknown time, possibly weeks from now, possibly while a source is blocking us.

**Choice.** A snapshot of real listings ships in `data/seed/` and loads on first boot. Crawling is a scheduled background job that enriches the data but is never on the request path.

**Why.** A demo that can fail is a demo that will fail, at the worst moment.

**Trade-off accepted.** Repo size, and a snapshot that ages. Refreshed before submission.

---

## ۶. کراولر دیوار از SorinFlow می‌آید — The Divar crawler is reused, not rewritten

**Context.** [SorinFlow](https://github.com/Tecso-Dev/SorinFlow-DaTA-mAmager) is my own production Divar collector — FastAPI + Playwright, running live on an Iranian VPS. Its scraper package is ~7,800 lines of code that has been maintained against a hostile, changing target.

**What transfers unchanged:**

| Module | Why it transfers |
|---|---|
| `stealth.py` | Browser args, context options and the stealth JS payload are category-agnostic |
| The `postlist/w/search` replay strategy | Divar's feed endpoint is the same for every category |
| DOM-scroll + cursor-pagination hybrid | Depth is the API's job, resilience is the DOM's; both independent of category |
| `normalize_persian_digits`, `parse_persian_number`, `parse_price_with_unit`, `extract_divar_id`, `parse_listing_card` | Persian text and price primitives, no real-estate assumptions |

**The single most valuable thing inherited** is a negative result verified on the wire on 2026-09-02: `GET api.divar.ir/v8/web-search/{city}/{category}` is dead. It returns **HTTP 200** with a `BLOCKING_VIEW` widget saying «نیاز به بروزرسانی», and a `last_post_date` of `-1` that is truthy in Python and poisons the next cursor. It looks like success and yields nothing. Starting fresh, that costs a day to discover.

**What does not transfer.** `extract_property_details`, `extract_rooms_from_text`, amenities and corner-type detection are real-estate specific and get replaced by car attribute extraction (brand, model, trim, year, mileage, body status).

**Choice.** Fork the collection layer, replace the parsing layer, keep the category slug configurable (`real-estate` → `car`).

**Trade-off accepted.** The crawler is Python/Playwright rather than Go. That moves a planned Go component into Python — which is the right call anyway: it is heavier on Playwright than on concurrency, and it widens the Python surface of this project, which is what the job description asks for. Go keeps the core API, ranker, cache, rate limiting and metrics.

---

## ۷. بدون احراز هویت، بدون شماره تماس — No authentication, no phone numbers

**Context.** SorinFlow also contains `auth.py`, `otp_store.py`, `contact_extractor.py` and `captcha_solver.py` — authenticated Divar sessions that extract advertiser phone numbers. That code works and it is right there.

**Choice.** None of it comes into KhodroBin. Collection is public, unauthenticated and rate-limited. No login, no OTP session, no CAPTCHA solving, no phone numbers. Where a stable seller identifier is needed to help decide that two ads are the same car, it is a salted SHA-256 of a public identifier, and the raw value is never persisted.

**Why.** Two reasons, and the first is sufficient on its own. Harvesting personal contact data for a public demo is not something to hand a company as a work sample. Second, price comparison does not need it: cross-listing identity is resolvable from normalized attributes and price proximity, which is the more interesting problem anyway.

**Trade-off accepted.** Identity resolution loses its strongest single signal and has to be genuinely good instead. Documented in `CRAWLING.md` alongside the rate limits, the bot User-Agent and the `robots.txt` policy.
