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

**Live at** `khodrobin.noxioai.com` (A → 95.179.189.5) and `khodro6.noxioai.com` (AAAA), both unproxied so Caddy can answer the ACME challenge, both on one certificate.

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
