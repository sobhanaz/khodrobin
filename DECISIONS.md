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

**Choice.** Go for the core API, ranking, caching, rate limiting and metrics. Python for the AI service — intent parsing, attribute extraction, explanation and the evaluation harness — and for the crawlers (see decision ۶). They talk over a small HTTP contract with a strict schema.

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

---

## ۸. زیرساخت دوپایه: کراولر در ایران، اپ در فرانکفورت — Split infrastructure

**Context.** Two hard constraints point in opposite directions, and neither is negotiable.

| Constraint | Consequence |
|---|---|
| Iranian marketplaces treat foreign datacenter IPs as hostile — throttling, CAPTCHA, outright blocks | The **crawler** wants an Iranian IP |
| OpenAI, Anthropic and Google block Iranian IPs outright (sanctions), and the block is on their side, so no amount of server config fixes it | The **AI service** cannot run on an Iranian IP |
| The reviewer opens the demo from Tehran | The **web tier** must be reachable from inside Iran |

A single box cannot satisfy all three. A Vercel deployment satisfies none of them — Vercel geo-blocks Iranian visitors at the edge, so the reviewer would see nothing.

**Options.**
- *All-Iranian VPS* — crawls perfectly, reviewer-friendly, but the AI layer is limited to local models only (Ollama).
- *All-foreign VPS* — model APIs work, but crawling Iranian sources is fragile and may fail silently at the worst moment.
- *Split.*

**Choice.** Split, with a deliberately thin seam.

```
┌────────────── Iran (VPS or the existing SorinFlow box) ──────────────┐
│  crawler (Playwright)  ──►  local queue  ──►  authenticated push      │
└───────────────────────────────────┬──────────────────────────────────┘
                                    │  POST /ingest  (mTLS or bearer, batched JSONL)
┌───────────────────────────────────▼──────────────────── Frankfurt ───┐
│  web (Nuxt) · api (Go) · ai (FastAPI) · postgres · elasticsearch      │
│  redis · caddy — reachable from Iran, reaches model APIs              │
└──────────────────────────────────────────────────────────────────────┘
```

The crawler is the only component that needs an Iranian IP, and it needs nothing from the internet except the three sources. Everything else moves to a Hetzner/Netcup box in Frankfurt, which can reach the model APIs and which Iranian users can reach normally — a plain VPS you control blocks nobody, unlike a CDN edge that geo-filters.

**Why this is worth saying on camera.** It is a real constraint that a Tehran-based reviewer will recognise instantly as true, and the response is an architectural decision rather than a workaround. «کراولر باید IP ایران داشته باشه چون منابع اجازه نمی‌دن؛ لایه‌ی مدل نمی‌تونه IP ایران داشته باشه چون API‌ها تحریم می‌کنن. پس جداشون کردم و با یک endpoint احراز‌هویت‌شده به هم وصلشون کردم.»

**Trade-off accepted.** Two machines, a network seam, and an ingest contract to secure and monitor. Mitigated by making ingest the *only* thing crossing the seam: batched, idempotent by content hash, authenticated, retried with backoff, and alerted on staleness. If the Iran box dies, the demo keeps serving from the last ingest — it degrades, it does not break.

**To verify on day 1.** Whether Divar and Bama actually serve a Frankfurt datacenter IP acceptably. `divar.ir` is reachable internationally; the open question is rate-limiting and CAPTCHA behaviour for foreign ASNs under sustained use. If they behave, collapse to a single Frankfurt box and delete this decision — one machine beats two. Do not assume it either way; measure it.

**Fallback if the model APIs stay out of reach.** Ollama with Qwen 2.5 7B, already a one-line provider switch. The product must work with a local model, and that is a design constraint, not a contingency.

---

## ۹. یک سرور، در فرانکفورت — One VPS, and it goes in Frankfurt

*Supersedes the two-box topology in decision ۸. The constraints there still hold; this is how they are satisfied with one machine.*

**Context.** One VPS is available, not two. Decision ۸ named three requirements that pull apart: the crawler wants an Iranian IP, the model APIs refuse Iranian IPs, and the reviewer opens the demo from Tehran.

**The observation that resolves it.** The crawler does not need to be on a server at all. It is a scheduled batch job that talks to three websites and pushes JSONL. It can run on my own machine in Iran — on a **residential** Iranian IP, which is a *better* crawling position than any Iranian datacenter IP, and far better than a foreign one.

**Choice.** The single VPS goes in **Frankfurt** and runs the entire product: Nuxt, the Go API, the FastAPI AI service, PostgreSQL, Elasticsearch, Redis, Caddy. The crawler runs locally in Iran on a schedule and pushes batches to an authenticated `/ingest` endpoint.

**Why Frankfurt and not Tehran.**

| | Frankfurt | Tehran |
|---|---|---|
| Reachable from Iran for the reviewer | ✅ a VPS you control geo-blocks nobody | ✅ |
| Model APIs (OpenAI / Anthropic / Gemini) | ✅ | ❌ blocked on their side; unfixable from ours |
| Crawling Iranian sources | handled from the Iranian residential IP | ✅ |
| Cost for 4 vCPU / 8 GB | ~€8/mo | more, for less |

Elasticsearch alone wants 1–2 GB, so 8 GB is the floor, not the target. Hetzner **CX32** or equivalent. Iranian resellers sell European boxes payable in Rial if a foreign card is a problem.

**Trade-off accepted.** Ingest depends on my machine being online. Three mitigations, all of which the product needs anyway: seed data ships in the repo so a cold visit always works; every record carries `last_seen_at` and the UI shows staleness honestly rather than pretending; and a staleness alert fires if no batch lands within N hours. The demo degrades toward "this data is 6 hours old" — it never shows an empty page.

**The honest framing for the video.** Not "I couldn't afford two servers", but: «کراولر روی IP خانگی ایران اجراست چون منابع ایرانی به IP دیتاسنتر خارجی جواب نمی‌دن، و اپ روی فرانکفورت چون API مدل‌ها از ایران در دسترس نیست. تنها چیزی که بین این دو رد و بدل می‌شه یک ingest احراز‌هویت‌شده‌ست.» That is a true constraint, met with a design, and every Iranian engineer watching will recognise it.

**Still to measure on day 1.** Whether Divar and Bama tolerate the Frankfurt IP under sustained use. If they do, the crawler moves onto the VPS and the local dependency disappears entirely. Measure before assuming — in either direction.

---

## ۱۰. یک سرور، همه‌چیز — measured, and the split was unnecessary

*Supersedes decisions ۸ and ۹. Both were designed around an assumption that turned out to be false.*

**Context.** Decisions ۸ and ۹ split the system because Iranian marketplaces were assumed to reject foreign datacenter IPs while model APIs reject Iranian ones. Decision ۹ ended with an instruction: *measure it on day 1, do not assume it either way.*

**The measurement.** From the Vultr Amsterdam box:

```
api.openai.com    = 401     model APIs reachable (401 = no key, not blocked)
api.anthropic.com = 405
divar.ir          = 200     Iranian sources reachable too
bama.ir           = 200
hamrah-mechanic   = 200
```

**Choice.** One server. No Iran box, no ingest seam, no dependency on my laptop being online. The crawler runs on the same machine as everything else.

**Why this is the better outcome.** The split design was defensible but it bought complexity to solve a problem that did not exist. Deleting it removes a network hop, an authentication surface, a staleness alert and an entire failure mode. The honest version of the video is now simpler and stronger: one box, boring stack, everything shipped from GitHub.

**What survives from ۸ and ۹.** The constraint itself is real and worth stating — a Tehran reviewer must be able to open the link, and model APIs will not answer an Iranian IP. Both are satisfied by putting the box in Amsterdam. And the caution stands: Divar *does* push back on sustained crawling, just not by IP geography — see decision ۱۲.

**Live at** `khodrobin.noxioai.com` (A) and `khodro6.noxioai.com` (AAAA), both unproxied so Caddy can answer the ACME challenge, both on one certificate.

---

## ۱۱. نه Vercel، نه Firebase — the demo is not hosted on anything that blocks Iran

**Context.** Both were considered for hosting the front end.

**The disqualifying fact.** Vercel is not reachable from Iran. Its upstream provider blocks OFAC-sanctioned countries at the infrastructure level and Vercel's own community threads describe it as outside their control. Firebase sits behind Google Cloud and falls in the same sanctions class. The reviewer for this application is in Tehran.

A link the reviewer cannot open is not a partial failure. It looks identical to having built nothing, and there is no way to find out that is what happened.

**Choice.** The submitted URL is always the VPS. Vercel may host a mirror and PR preview deployments for convenience while building — that is real value for review workflow — but it never appears in the README, the video, or the submission form.

**Firebase is declined outright**, for a second reason beyond reachability: this product is relational. Market medians are `GROUP BY spec_key` over offers. Deduplication is a join. Postgres is the right tool and it is already running on a box with 15 GB of RAM doing nothing. Adding a document store would mean a second source of truth to keep in sync, for no capability we lack.

**Trade-off accepted.** No edge CDN in front of the app. At the traffic a hiring demo sees, Caddy with zstd on an 8-vCPU box is not the bottleneck, and Cloudflare can be put in front later without touching the application.

---

## ۱۲. اعتبارسنجی قبل از استقرار — validate config before it can ship

**Context.** A Caddyfile with a single-line `handle /healthz { reverse_proxy api:8080 }` block is invalid syntax. Caddy rejected it and crash-looped, while Postgres, Redis and the API all reported healthy. Nothing failed loudly; the only signal was the smoke test failing three minutes into the deploy.

**Choice.** CI runs `caddy validate` on every push. Config is code and gets the same gate the code gets.

**Why it is worth a decision entry.** The failure mode is the interesting part: every health check was green while the thing in front of them was dead. Liveness probes on components tell you nothing about whether the system answers, which is exactly why the deploy ends with an external smoke test against the public URL rather than a `docker compose ps`.

---

## ۱۳. واحد مقایسه، «مشخصات» است نه یک خودروی مشخص — the unit of comparison

**Context.** The original plan was to detect that three ads on three sites are the *same physical car* and show one row with three prices. I checked whether that was true of the data before building it.

**The measurement.** It is not. In 1,875 listings the cross-source "collisions" were different cars at similar prices — a KMC X5 next to a Daiyun Y7. Physical duplicates across marketplaces are rare enough that no honest volume of crawling would surface them reliably.

**Choice.** The unit is the **spec**: brand, model, trim, gearbox, year, mileage band — with every matching offer underneath and a real price distribution.

**Why this is better, not a retreat.** It is what Torob's own card already is. Their page shows one *product* with «در X فروشگاه», not one physical object. A used car has no SKU, so building that key requires inferring `{brand, model, trim}` from three incompatible vocabularies — which is precisely «استخراج ویژگی‌های محصول» plus «تشخیص یکسان بودن کالاها», their two hardest listed problems.

**Trade-off accepted.** The demo cannot claim "this exact car is cheaper over there". It claims something more useful and more defensible: "this spec has a median of X across N offers from M marketplaces, and this one is 12% under it".

---

## ۱۴. منبع چهارم: خودرو۴۵ — a fourth source, and why it changed nothing at first

**Context.** Khodro45 was added as a fourth marketplace. Its API is the cleanest of the four: DRF pagination, canonical slugs, an explicit trim, and `is_klm_matched` — whether the odometer was verified, which no other source offers.

**What happened.** Adding 240 listings moved multi-source clusters from 66 to 66. Exactly nothing.

**The cause.** Khodro45 has no transmission field; it files the gearbox inside `trim` («اتوماتیک» ×67, «دنده‌ای» ×26). Since gearbox is part of the spec key, every Khodro45 listing keyed on `gearbox=na` and could never join another source's cluster. It silently formed a parallel universe of single-source specs.

**With it parsed out:** 66 → 92 multi-source clusters, 2 → 14 with three or more sources, 39 clusters now including Khodro45.

**The general lesson, and it is the second time:** Hamrah put the gearbox in the same field as the trim, and so does Khodro45. Any field that mixes two dimensions will silently silo the data rather than fail loudly. Worth a test each time, which is what `test_khodro45_gearbox_is_read_out_of_the_trim_field` is for.

---

## ۱۵. pnpm به جای npm — npm cannot resolve the Nuxt tree

**Context.** The web image would not build: `npm ci` rejected a lockfile `npm install` had just written.

**The root cause is upstream of that.** `npm install --package-lock-only` itself crashes with `Cannot read properties of null (reading 'edgesOut')`. Bisected: it reproduces with `nuxt + vue + vue-router` alone — no Tailwind, no TypeScript — under npm 10 in `node:22-alpine` and npm 11 locally. It is an arborist bug, not a dependency conflict we can pin our way out of.

**Choice.** pnpm, which resolves the same `package.json` in 1.4 seconds, is what Nuxt recommends, and gives CI real reproducibility through `--frozen-lockfile`.

**Detail worth keeping.** The lockfile is generated inside `node:22-alpine`, the same image that installs from it. Generating it with a different runtime than the one that consumes it is how the original mismatch appeared in the first place.

---

## ۱۶. حساب کاربری فقط برای ذخیره و هشدار — accounts, and why search stays anonymous

**Context.** Decision ۷ said this product needs no user accounts, and for pure search that was right. The scope later grew to include registration, an admin panel and a landing page.

**The question that decides it.** Not "do we want accounts" but "what does an account let a person do that they cannot do already". A login that only unlocks the thing anonymous visitors already have is friction wearing a feature's clothes.

**Choice.** Accounts exist for **saved searches and price alerts**, and nothing else. Search, ranking, explanations and the per-car pages stay fully anonymous and are never gated.

**Why that specific pair.** Torob has «پیگیری قیمت». A used-car price that moves is exactly the thing a buyer wants told to them rather than having to come back and check, and telling them requires knowing where to write. That is a real reason to hold an address; "so we have users" is not.

**Trade-off accepted.** A database, a mail dependency, session handling and an authentication surface — all of which have to be got right, and none of which the search product needed. Mitigated by keeping the blast radius small: the auth service is separate, has its own Caddy prefix, and an outage in it cannot take search down.

**Consequences worth stating, because they were deliberate:**
- The signup form never reveals whether an address is registered, and login hashes a password even for accounts that do not exist so response time cannot become the same oracle.
- Access tokens live in memory, refresh tokens in an HttpOnly cookie. An XSS bug then borrows a session for fifteen minutes rather than stealing one for thirty days.
- Refresh-token reuse is treated as compromise and revokes every session for that user. Annoying once, versus an attacker holding a session for a month.
- The admin route answers 404 to a non-admin, because 403 confirms both that the route exists and that this account is merely not privileged enough.
- Arming a price alert requires a verified address. Otherwise the product is a way to send mail to arbitrary strangers.
- `is_admin` defaults to false with no path to raise it from the application. Granting it is a deliberate SQL statement, not a feature.

---

## ۱۷. صفحه‌ی هر خودرو برای موتور جست‌وجو — server-rendered pages, and the markup that made this project possible

**Context.** The product had one page. A search product with one page is invisible to search engines.

**Choice.** Every spec gets a server-rendered page at `/car/{key}` publishing `schema.org/Car` structured data, plus a sitemap generated from the live index and a generated `robots.txt`.

**Why it is on-strategy rather than bolted on.** This project can crawl Divar without a browser precisely *because* Divar publishes schema.org JSON-LD for search engines. Publishing the same markup is not imitation; it is the same mechanism, and it is what makes a catalogue legible to anything other than a human with a mouse.

**Details that decide whether the markup is worth anything:**
- schema.org has no toman. Publishing toman figures under `priceCurrency: IRR` would understate every price tenfold, so the numbers are converted to the unit the code actually names.
- The sitemap is generated, not stored. The catalogue changes every three hours; a file would be wrong almost immediately.
- Specs are ordered by corroboration. A crawl budget is finite and a single-offer card is the least useful page here.
- A sitemap that 500s teaches a crawler to stop asking, so a failed index fetch still returns valid XML with the static pages listed.
- Result cards link to their own page. Without a link there is nothing to follow.

**Trade-off accepted.** Thousands of thin pages if the index grows and most specs stay single-source. The corroboration ordering is the current answer; a `noindex` on single-offer specs is the next one if it becomes a problem.

---

## ۱۸. پرچم تزئینی نیست — a flag that does not change behaviour is decoration

**Context.** Outlier detection worked: a Saina listed at 82% under its cohort was correctly tagged `price_outlier`, and its title was «فروش و مشارکت حواله ساینا» — an allocation certificate, not a car.

**What happened anyway.** It was still the cheapest number, so it still sorted first, and the model was still handed it as the thing to justify. The live card explained itself as «۸۱.۹٪ کمتر از میانه‌ی بازار».

**The part worth remembering.** The hallucination guard could not catch this and should not have. 81.9٪ was the true figure. **An explanation can be perfectly accurate and still describe the wrong thing** — accuracy is a property of a sentence, relevance is a property of what you chose to say it about.

**Choice.** Offers sort flagged-last within a spec, and flagged offers are removed before the top three reach the model — falling back to the raw list only when every offer is flagged, because some explanation beats none.

**Trade-off accepted.** A genuinely cheap listing that trips a flag gets buried. That is the right direction to be wrong in: showing a real bargain second costs a user one scroll, while leading with a حواله costs them their trust in every number on the page.

---

## ۱۹. راست بودن کافی نیست — a true sentence can still be nonsense

**Context.** Four guard axes all asked the same question in different ways: *did the model invent this?* Numbers, percentages, topics and sources are each a way of catching a fabrication.

**What got through.** A live card read «این خودرو با قیمت ۸۲۸,۰۰۰,۰۰۰ تومان، ۵۵.۷٪ زیر میانه‌ی بازار است. در عوض، کارکرد صفر کیلومتر را **از دست می‌دهی**» — *in exchange, you lose the zero kilometres.* The price is real. The percentage is real. The mileage is real. All four axes passed it, correctly, and the sentence is still absurd — it names the top offer's best feature as the thing you sacrifice by choosing it.

**Why it happens, structurally.** The prompt asks the second sentence to name a trade-off. When the cheapest offer is also the newest with the lowest mileage, there is no trade-off to name — and a model instructed to produce one produces one anyway. Measured across the live index: **10% of model explanations used loss framing**, several pointed at a dimension the top offer led. The system prompt already warns against exactly this; a prompt is a request.

**Choice.** A fifth axis. Loss framing («از دست می‌دهی», «چشم‌پوشی») is rejected when it lands on a dimension where the top offer is the best of its cohort. The fallback — assembled from data, so coherent by construction — takes over.

**Trade-off accepted.** It is a phrase list, so it is Persian-specific and incomplete; a model can express the same idea in words it does not contain. It catches the failure that actually occurred at the rate it actually occurred, which beats a general solution that does not exist.

**The general form.** *Accuracy and coherence are independent properties, and only one of them was being checked.* Compare ۱۸: an explanation can also be accurate and coherent and still be about the wrong thing.

---

## ۲۰. نسخه‌ای که کسی باید یادش باشد، نسخه نیست — a version constant is not a mechanism

**Context.** After adding the fifth axis, nothing changed for visitors. The guard would now reject 13 of 93 live explanations, and all 93 kept being served.

**Why.** Two caches — the AI service's in-memory LRU and the warm file the crawler writes every three hours — both key on the **data**. Prices had not moved, so every entry was considered fresh. Both had a knob for exactly this case: `PROMPT_VERSION`, sitting at `"1"`, with a comment saying to bump it when the prompt changed. The prompt and the guard had changed a dozen times. It had never been bumped, and nothing had ever noticed.

**Choice.** `PROMPT_VERSION` is now a SHA-256 of the guard module's own source, published on `/health` and folded into the warm fingerprint. Editing the guard expires everything the old guard approved, in both caches, without anyone deciding to.

**Trade-off accepted.** It is coarse — a comment change invalidates 149 explanations and costs ~25 minutes of background regeneration inside a 1800s budget. That is the cheap resource in this architecture, and the precise alternative (re-validating each cached text) needs the guard inside the crawler image, which builds from `services/crawler/` alone. Wrong resource to optimise.

**The general form.** *A safety mechanism that depends on someone remembering is documentation, not a mechanism.* The failure mode is silence — it never errors, it just quietly stops being true.
