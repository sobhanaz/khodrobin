# KhodroBin — Production & Submission Checklist

Status key: ✅ verified done · ◐ partial / uncommitted · ⬜ missing · 🚫 deliberately skipped (write-don't-build)

## 1. The demo & the win (highest leverage)

- [ ] ✅ All 5 rubric lines map to modules in README
- [ ] 🚧 Demo video recorded (shot list: `docs/ROADMAP.md` §19) — the only unmet rubric line
- [ ] ⬜ Reverse-lookup wow: paste any listing URL → same car across all sources → overpay verdict
- [ ] ⬜ Live guard demo: hostile ad title rejected on camera
- [ ] ◐ Haggle-message generator (only if video days allow)
- [ ] ⬜ Test live URL from an Irancell/MCI phone, Wi-Fi off — the single most important check

## 2. Frontend / UI-UX

- [ ] ✅ Design tokens in `main.css` (bg/surface/ink/accent/good/warn/focus)
- [ ] ✅ Persian type scale (line-height ≥1.8 body, zero negative tracking)
- [ ] ✅ Glass only for chrome, solid behind data
- [ ] ✅ RTL-native (`dir=rtl` at html level), logical properties
- [ ] ✅ Vazirmatn self-hosted (no Google Fonts — Iran-unreachable)
- [ ] ✅ `prefers-reduced-motion` / `prefers-reduced-transparency`
- [ ] ✅ Focus ring color ≠ error color (deliberate, documented)
- [ ] ◐ Landing redesign — uncommitted, never reviewed
- [ ] ◐ New verify-page OTP form — needs one visual pass
- [ ] ⬜ Empty states (no results, errors) get real visual design
- [ ] ⬜ Lighthouse ≥90 perf + a11y, screenshot into README
- [ ] ⬜ `nuxt typecheck` wired into CI (needs generated tsconfig)

## 3. Backend / API (Go)

- [ ] ✅ Search ~1ms, parse ~0.01ms, rank ~1.4ms (measured, in README)
- [ ] ✅ Intent parsing + deterministic fallback
- [ ] ✅ 4 ranking modes with per-factor score breakdown
- [ ] ✅ Index hot-reload without restart; seed fallback baked into image
- [ ] ✅ `/healthz` `/readyz` `/api/v1/stats`
- [ ] ⬜ `internal/ratelimit/` — empty; build token-bucket limiter (429 + Retry-After + X-RateLimit-*)
- [ ] ⬜ `internal/metrics/` — empty; expose Prometheus `/metrics`
- [ ] ⬜ `internal/cache/` — empty; either in-process SWR cache with `X-Cache` header, or delete the dir
- [ ] ⬜ `internal/rank/`, `internal/store/` — empty dirs; delete or fill (ghosts are worse than absence)
- [ ] ⬜ Reverse-lookup endpoint (`POST /api/v1/lookup`) for the wow feature

## 4. AI service (FastAPI)

- [ ] ✅ Five-axis hallucination guard (numbers, percentages, topics, sources, coherence)
- [ ] ✅ Templated fallback that always passes the guard
- [ ] ✅ Cache keyed on a hash of the guard's own source (self-invalidating)
- [ ] ✅ Daily cost ceiling ($5), `/metrics`, warm explanations (149 precomputed)
- [ ] ✅ Eval harness gating CI (literal 50/50, hard 30/30, messy 10/12)
- [ ] ⬜ Nightly eval cron + accuracy-regression Telegram alert

## 5. Auth / accounts (Go + Postgres)

- [ ] ✅ Enumeration-safe register/login/reset/forgot (same reply for known/unknown)
- [ ] ✅ Timing-safe login (hashes password for nonexistent accounts)
- [ ] ✅ Rotating refresh tokens, reuse = revoke every session
- [ ] ✅ Access in memory, refresh in HttpOnly cookie (XSS borrows 15 min, not 30 days)
- [ ] ✅ Admin route answers 404 not 403; argon2id; failed-login lockout persisted
- [ ] ✅ Saved searches + price alerts; alerts require verified address
- [ ] ✅ Newsletter double opt-in, idempotent unsubscribe, mailable export filters in SQL
- [ ] ✅ Email redesign + OTP code flow (link AND 6-digit code; welcome / password-changed / alert-confirmation mails)
- [ ] ◐ All of the above is uncommitted
- [ ] ⬜ SPF / DKIM / DMARC on the domain (verification mail currently lands in spam)
- [ ] ⬜ Admin account granted (one SQL statement)
- [ ] ⬜ Change-password-while-logged-in endpoint (only reset exists)

## 6. Crawler / data (Python)

- [ ] ✅ 5 sources plain HTTP (Divar, Bama, Hamrah-Mechanic, Khodro45, Sheypoor ◐), no browser automation
- [ ] ✅ Politeness: ≤1 req/s, jitter, robots.txt, circuit breaker, yield-measuring (empty 200 = soft block)
- [ ] ✅ Unit reconciliation (rial/toman, Jalali/Gregorian) — verified by pricing one car across sources
- [ ] ✅ Dedupe (40% re-crawls collapsed), plausibility, contradiction + outlier flags shown not repaired
- [ ] ✅ Median suppressed below 3 offers
- [ ] ⬜ Price-history snapshot `{spec_key, date, median_price, offer_count}` on each 3h crawl
- [ ] ⬜ `CRAWLING.md` (rate limits + why — a culture signal to Torob)

## 7. CDN / edge / DNS

- [ ] ✅ Caddy: zstd/gzip, HSTS, nosniff, Referrer-Policy, Permissions-Policy, `-Server`
- [ ] ◐ Cloudflare DNS-only by choice (no proxy/CDN) — document why; fine at this scale
- [ ] ⬜ CSP (`default-src 'self'`) in Caddyfile
- [ ] ⬜ og:image / Twitter card per spec page (links pasted in Telegram show a blank card)
- [ ] ⬜ A/AAAA tested from Iranian networks (v6 has its own name already)

## 8. Docker / containers

- [ ] ✅ distroless nonroot for Go services; multi-stage everywhere
- [ ] ✅ Index baked into api image; seed fallback
- [ ] ✅ Ollama internal-only; model pulled idempotently on boot
- [ ] ✅ Healthchecks + `depends_on: service_healthy` (dev compose)
- [ ] ◐ Prod compose: web/ai/auth/crawler/caddy have **no healthcheck or resource limits**
- [ ] ⬜ Base images pinned by digest; `ollama:latest` → versioned tag
- [ ] ⬜ `read_only` rootfs + `cap_drop: [ALL]` on api/auth
- [ ] ⬜ CI job that docker-builds all 5 images (breakage caught at PR time)

## 9. CI/CD

- [ ] ✅ gofmt/vet/race, pytest, Nuxt build, Caddyfile validation, eval gate, auth tests
- [ ] ✅ Deploy: GHCR images, tag + latest, `chmod 600` .env on box, caddy reload, real smoke test (readyz + homepage + query hits)
- [ ] ⬜ golangci-lint, ruff, eslint/vue-tsc jobs
- [ ] ⬜ trivy (fail HIGH/CRITICAL) + govulncheck + pip-audit
- [ ] ⬜ Rollback on failed smoke test (pull previous tag)
- [ ] ⬜ Dependabot/Renovate weekly

## 10. Security

- [ ] ✅ PII: seller phones hashed with salt, raw never persisted
- [ ] ✅ SQL parameterised everywhere; CSV export defuses formula injection
- [ ] ✅ No CORS surface (same origin, split at Caddy edge)
- [ ] ⬜ CSP header (see §7)
- [ ] ⬜ Inbound rate limiting (see §3)
- [ ] ⬜ Rotate the root password shared in chat 6 Sep
- [ ] ⬜ Container/image scans in CI (see §9)

## 11. Observability / monitoring

- [ ] ✅ AI `/metrics` (spend, calls, cache hits, guard rejections, fallbacks)
- [ ] ⬜ API `/metrics` (see §3)
- [ ] ⬜ Prometheus scraping anything at all (currently nothing does)
- [ ] ⬜ Alerts → Telegram: source-down, cost-spike, p95, accuracy regression, disk
- [ ] ⬜ Nightly eval cron (see §4)
- [ ] 🚫 Tracing, Loki, Grafana — write in SCALING.md, don't build

## 12. Data / backups / DR

- [ ] ✅ Idempotent migrations on boot; pgxpool; named volumes; Redis AOF
- [ ] ⬜ Nightly `pg_dump` to second disk + off-box
- [ ] ⬜ One tested restore (screenshot the evidence)
- [ ] 🚫 Read replicas / partitioning — SCALING.md triggers only

## 13. Email / communication

- [ ] ✅ 7 transactional templates, one theme, preheader + color-scheme + fallback link
- [ ] ✅ OTP + link dual path; welcome on first verify only; password-changed on reset; alert confirmation
- [ ] ✅ Double opt-in; suppression honored; no promo mail without consent
- [ ] ⬜ SPF/DKIM/DMARC (see §5)
- [ ] 🚫 Bounce handling / send analytics — skip at this scale

## 14. Performance

- [ ] ✅ Real measured numbers in README (parse/rank, dedupe %, eval %, $/query)
- [ ] ⬜ k6 load test (50 VUs, 2 min) → publish p95
- [ ] 🚫 CDN, multi-region — SCALING.md

## 15. SEO / growth

- [ ] ✅ Per-car SSR pages with `schema.org/Car`
- [ ] ✅ Generated sitemap + robots from live index
- [ ] ⬜ og:image per spec (see §7)
- [ ] 🚧 Market report pages / price index — wow backlog, only if time

## 16. Legal / privacy

- [ ] ✅ PII posture documented (hash, no raw phone)
- [ ] ⬜ `/privacy` page: what we store, what we hash, retention, delete-me path
- [ ] ⬜ `/terms` (short)

## 17. Docs / repo

- [ ] ✅ README: rubric table, eval block, guard story, architecture, Iran-reachability, seed
- [ ] ✅ DECISIONS.md, HANDOFF.md (now with session notes + roadmap + wow backlog)
- [ ] ⬜ `CRAWLING.md` (see §6)
- [ ] ⬜ `SCALING.md` (100× table with triggers)
- [ ] ◐ Everything uncommitted — commit the tree
- [ ] ✅ ~106 tests across 5 services, eval thresholds gated in CI

## 18. Final pre-submission run

- [ ] Commit everything; CI green end-to-end
- [ ] Fresh `make eval` numbers into README
- [ ] Lighthouse screenshot, k6 numbers, backup-restore evidence into README
- [ ] Record + edit video (≤4:50) around the reverse-lookup moment
- [ ] Upload to Aparat (primary) + unlisted YouTube
- [ ] Test live URL: incognito desktop + **Iranian mobile data, Wi-Fi off**
- [ ] Submit both links labelled: primary (Iran-reachable) + mirror