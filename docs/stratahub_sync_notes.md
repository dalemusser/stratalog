# Changes in stratalog to port to strata_hub

## TLS / DNS-01 support (Route53 + lego)
- File: `internal/app/system/config/config.go`
  - Added fields:
    - `lets_encrypt_challenge` (`"http-01"` | `"dns-01"`)
    - `route53_hosted_zone_id`
- File: `internal/app/system/server/server.go`
  - Added DNS-01 branch in `StartServerWithContext` switch:
    - Case: `use_lets_encrypt=true` AND `lets_encrypt_challenge="dns-01"`
    - Does **not** start an `:80` aux server (DNS-only validation; no HTTP-01).
  - Uses `obtainOrLoadDNSCert(cfg)` to load/generate certs (see `dnslego.go`).
- File: `internal/app/system/server/dnslego.go`
  - New helper for lego + Route53 DNS-01:
    - Uses lego’s Route53 DNS provider to:
      - Register ACME account (if needed)
      - Solve DNS-01 challenges
      - Obtain cert for `cfg.Domain`
      - Cache cert + key under `lets_encrypt_cache_dir`.
> Note: DNS-01 mode does not require port 80 and allows HTTPS to run on non-standard ports (e.g., 8443) without HTTP-01 challenges.

---

## Prometheus metrics
- File: `internal/app/system/metrics/metrics.go`
  - Added HTTP histogram (`http_request_duration_seconds`) with labels:
    - `path`, `method`, `status`
  - Added registration of default Go/runtime & process collectors using:
    - `collectors.NewGoCollector`
    - `collectors.NewProcessCollector`
  - Made registration idempotent (ignores `AlreadyRegisteredError`).
- File: `internal/app/system/routes/routes.go`
  - Mounted `/metrics` using `metrics.MetricsRoute()`.
- File: `cmd/stratalog/main.go`
  - Calls:
    - `metrics.RegisterDefaultPrometheus()`
    - `metrics.MustRegisterMetrics()`
- File: `internal/app/system/server/server.go`
  - Added `appmetrics.Metrics` middleware to chi router:
    - Placed after panic recovery and before CORS/compression/logging.

---

## Config & API key auth
- File: `internal/app/system/config/config.go`
  - For stratalog:
    - Changed env prefix to `STRATALOG` (module-specific).
    - Default `mongo_database` set to `"stratalog"`.
    - Removed hub-specific fields:
      - `session_key`
      - CloudFront URL
      - Google OAuth fields (`google_key`, `google_secret`, `google_callback`)
  - Added security fields:
    - `ingest_api_key` (required): protects log ingestion endpoints.
    - `admin_api_key` (optional): for admin/dev endpoints (`/logs/view`, `/logs/download`, `/metrics`, `/debug/pprof`).
  - Validation:
    - Require `ingest_api_key` (env: `STRATALOG_INGEST_API_KEY` or `--ingest_api_key`).
    - Extended TLS validation for new DNS-01 challenge mode.
> For strata_hub:
> - Keep the env prefix as `STRATA_HUB`.
> - Keep `session_key` and Google OAuth fields.
> - Only copy the new fields (API keys, DNS-01 config, request body limit) and associated validation, not the removal of session/OAuth behavior.

---

## Main / server cleanup & bootstrap
- File: `cmd/stratalog/main.go`
  - Removed `timezones.Load()` step (stratalog does not need time zone catalog).
  - Kept the staged startup:
    - bootstrap logger → load config → build final logger →
      connect Mongo → validators → indexes → template engine → start HTTP server.
- File: `internal/app/system/server/server.go`
  - Removed session/OAuth initialization (`auth.InitSessionStore`, Google provider) for this app.
    - stratalog uses only API-key auth at the HTTP layer.
  - Retained:
    - CORS-from-config
    - Brotli + gzip compression
    - Zap access logging
    - Structured HTTP timeouts & graceful shutdown.

---

## Health endpoint and shared view layout bootstrap
- File: `internal/app/features/health/health.go` and `routes.go`
  - Mounted `/health` endpoint that:
    - Pings Mongo with a 2s timeout.
    - Returns JSON:
      - `{"status":"ok","database":"connected"}` on success.
      - `{"status":"error","message":"Database unavailable","error":"..."}` with 503 on failure.
- File: `internal/app/features/shared/views/views.go`
  - Registered `shared` template set using `embed.FS`.
- File: `internal/app/features/shared/views/templates/layout.gohtml`
  - Minimal shared layout:
    - Loads Tailwind + HTMX.
    - Defines `"layout"` and `"content"` blocks used by other features.
> For strata_hub: health and shared layout already exist; use this as a pattern reference rather than a direct port, unless response formats or templates need aligning.

---

## API Key Middleware (ingest + admin keys)
- New package: `internal/app/system/apikey/apikey.go`
  - Provides two middlewares:
    - `RequireIngestKey(cfg)` — protects log ingestion endpoints.
      - Accepts keys via:
        - `Authorization: Bearer <key>`
        - `X-API-Key: <key>`
        - `?api_key=<key>` (for browser/dev use).
    - `RequireAdminKey(cfg)` — protects admin/dev endpoints.
      - Behavior:
        - If `admin_api_key` is set: only that key is accepted.
        - If `admin_api_key` is empty: falls back to `ingest_api_key`.
  - Includes helper `apiKeyFromRequest(r)` to normalize key extraction.
- File: `internal/app/system/config/config.go`
  - Added config fields:
    - `ingest_api_key` (required).
    - `admin_api_key` (optional; fallback to ingest key when empty).
  - Added validation requiring `ingest_api_key`.
> For strata_hub:
> - stratalog removed session-based auth entirely; strata_hub keeps session auth.
> - Only the **API-key middleware + config fields** should be ported, not the removal of sessions.
> - For hub, API keys may augment rather than replace session/role checks.

---

## HTTP utilities & unified JSON errors
- New package: `internal/platform/httputil/json.go`
  - Provides helpers:
    - `WriteJSON(w, status, v)` — sets `Content-Type: application/json`, writes status and encodes payload.
    - `JSONError(w, status, code, message)` — writes `{"error":"<code>","message":"<message>"}`.
    - `JSONErrorSimple(w, status, message)` — writes `{"error":"<message>"}`.
  - Logs JSON encoding failures via Zap.
- File: `internal/app/features/health/health.go`
  - Updated to use `httputil.WriteJSON` and `httputil.JSONError` for consistent JSON responses.

---

## pprof debugging endpoints
- File: `internal/app/system/routes/routes.go`
  - Added `mountPprof(r chi.Router)` helper that wires Go's standard pprof handlers under `/debug/pprof`:
    - `/debug/pprof/` — index
    - `/debug/pprof/cmdline`
    - `/debug/pprof/profile`
    - `/debug/pprof/symbol` (GET/POST)
    - `/debug/pprof/trace`
    - `/debug/pprof/{name}` — e.g. `goroutine`, `heap`, etc.
  - Called `mountPprof(r)` from `RegisterAllRoutes`.
> For strata_hub: wiring pattern is the same; see also the protected group below.

---

## Protect /metrics and /debug/pprof with API key
- File: `internal/app/system/routes/routes.go`
  - Updated `RegisterAllRoutes`:
    - `health.MountRoutes(r, h)` and `version.MountRoutes(r, h)` remain public.
    - Wrapped `/metrics` and `/debug/pprof/*` in:
      ```go
      r.Group(func(r chi.Router) {
          r.Use(apikey.RequireAdminKey(h.Cfg))
          r.Handle("/metrics", metrics.MetricsRoute())
          mountPprof(r)
      })
      ```
- Behavior:
  - `RequireAdminKey` uses `admin_api_key` if set, otherwise falls back to `ingest_api_key`.
  - Accepted credentials:
    - `Authorization: Bearer <key>`
    - `X-API-Key: <key>`
    - `?api_key=<key>`
> For strata_hub: `/metrics` and `/debug/pprof` should also be protected with admin-level auth. In hub this will likely use session/role checks (e.g., `RequireAdmin`) rather than API-key middleware, or a combination of both.

---

## Request size limit (config + middleware)
- New middleware: `internal/platform/httputil/limit.go`
  - `LimitBodySize(maxBytes int64) func(http.Handler) http.Handler`
    - Wraps `r.Body` with `http.MaxBytesReader(w, r.Body, maxBytes)`.
    - Prevents handlers from reading more than `maxBytes` from the request body.
    - If the limit is exceeded, subsequent reads fail and decoders see an error.
- File: `internal/app/system/config/config.go`
  - Added field: `max_request_body_bytes` (int64).
  - Added flag: `--max_request_body_bytes` (bytes; `0` = unlimited).
  - Defaults: `max_request_body_bytes = 2<<20` (2 MiB).
  - Validation: must be `>= 0`.
- File: `internal/app/system/server/server.go`
  - Now uses `cfg.MaxRequestBodyBytes`:
    - `r.Use(httputil.LimitBodySize(cfg.MaxRequestBodyBytes))`
> For strata_hub: same pattern can be used to protect form submissions, CSV uploads, etc., or adjusted per route if needed.

---

## Improved HTTP access logging (zap)
- File: `internal/app/system/server/mw_zaplogger.go`
  - Logs additional fields:
    - `host`   — value of `r.Host`
    - `scheme` — derived from `r.TLS` or `X-Forwarded-Proto`
    - `proto`  — HTTP protocol version
    - `route`  — chi route pattern (e.g. `/health`, `/metrics`, `/debug/pprof/{name}`)
  - Keeps existing fields:
    - `method`, `path`, `status`, `bytes`, `remote_ip`, `user_agent`, `referer`, `latency`, `request_id`
> For strata_hub: same improved logger should be ported so all services share a consistent, richer HTTP log format.

---

## Startup configuration summary logging
- File: `cmd/stratalog/main.go`
  - Added helper `logStartupSummary(cfg *config.Config)` that logs a structured startup summary using zap:
    - `env`, `log_level`
    - `http_port`, `https_port`, `use_https`, `use_lets_encrypt`
    - `tls_mode` (`http-only` | `LE http-01` | `LE dns-01` | `manual`)
    - `domain`, `lets_encrypt_challenge`, `lets_encrypt_cache_dir`
    - `mongo_uri` (redacted), `mongo_database`, `index_boot_timeout`
    - `max_request_body_bytes` (raw + human)
    - `enable_compression`, `enable_cors`, CORS list sizes (origins/methods/headers)
    - `has_ingest_api_key`, `has_admin_api_key` (booleans; values not logged)
    - `health`, `metrics`, and `pprof` endpoints (`/health`, `/metrics`, `/debug/pprof`)
  - Called `logStartupSummary(cfg)` after final logger initialization so each process logs its effective runtime configuration once at startup.
  - Startup summary now logs a redacted Mongo URI:
    - Uses `net/url` to strip any password from the userinfo before logging.
    - Ensures DB credentials never appear in logs.
> For strata_hub: port this function and adapt fields to hub's config (session_key presence, auth providers, etc.) so all services emit a consistent startup summary.

---

## Zap-based panic recovery with stack traces
- New file: `internal/app/system/server/mw_recover.go`
  - Added `zapRecoverer(l *zap.Logger) func(http.Handler) http.Handler`:
    - Wraps each request in a `defer` that recovers from panics.
    - Logs:
      - normalized error (`zap.Error(err)`)
      - raw panic value (`panic_value`)
      - stack trace (`debug.Stack()` as `stacktrace`)
      - `method`, `path`, `remote_ip`, `request_id`
    - Responds with HTTP 500 on panic.
- File: `internal/app/system/server/server.go`
  - Replaced chi's `middleware.Recoverer` with:
    - `r.Use(zapRecoverer(zap.L()))`
  - Ensures all HTTP panics are logged via zap with full stack traces and request context.
> For strata_hub: replace `middleware.Recoverer` with the same `zapRecoverer` to unify panic logging across services.

---

## Version endpoint (/version)
- New package: `internal/app/system/versioninfo`
  - `versioninfo.Service`   — service name (default `"stratalog"`)
  - `versioninfo.Version`   — semantic version or build tag (default `"dev"`)
  - `versioninfo.GitCommit` — git commit SHA (default `""`)
  - `versioninfo.BuildTime` — build timestamp (default `""`)
  - Fields intended to be overridden via `-ldflags` at build time.
- New feature: `internal/app/features/version`
  - `version.MountRoutes(r, h)` registers:
    - `GET /version` — returns JSON:
      - `service`, `version`, `git_commit`, `build_time`, `go_version`
  - Uses `httputil.WriteJSON` for consistent JSON responses.
- File: `internal/app/system/routes/routes.go`
  - Updated `RegisterAllRoutes` to mount:
    - `health.MountRoutes(r, h)` (public)
    - `version.MountRoutes(r, h)` (public)
    - `/metrics` and `/debug/pprof/*` remain protected via `RequireAdminKey`.

---

## Content-Type guard & unified 404/405 handling
- New middleware: `internal/platform/httputil/contenttype.go`
  - `RequireJSON()`:
    - Ensures `Content-Type` is `application/json` or ends with `+json`.
    - On mismatch, returns `415 Unsupported Media Type` with JSON error:
      - `{"error":"unsupported_media_type","message":"Content-Type must be application/json"}`.
- New handlers: `internal/platform/httputil/handlers.go`
  - `NotFoundHandler(w, r)`:
    - Logs `not_found` with method/path/remote_ip.
    - Returns 404 with JSON body:
      - `{"error":"not_found","message":"The requested resource was not found"}`.
  - `MethodNotAllowedHandler(w, r)`:
    - Logs `method_not_allowed` with method/path/remote_ip.
    - Returns 405 with JSON body:
      - `{"error":"method_not_allowed","message":"The requested HTTP method is not allowed for this resource"}`.
- File: `internal/app/system/server/server.go`
  - After creating the router:
    - `r.NotFound(httputil.NotFoundHandler)`
    - `r.MethodNotAllowed(httputil.MethodNotAllowedHandler)`
  - (Future) `/logs` JSON POST endpoints will also use `httputil.RequireJSON()` in their route groups.

---

## Redacted MongoDB URI in startup logs + DRY helper
- File: `cmd/stratalog/main.go`
  - Updated Step 4 (MongoDB connection) and startup summary to avoid logging credentials.
  - Introduced helper: `redactMongoURI(raw string) string`
    - Parses the Mongo URI with `net/url`.
    - If userinfo is present, preserves the username and replaces the password with `"****"`.
    - Returns the redacted URI string; on parse errors or no userinfo, returns the original URI.
  - Updated:
    - `logStartupSummary(cfg)` to use `redactMongoURI(cfg.MongoURI)` for the `mongo_uri` field.
    - Step 4 ("Connect to MongoDB") to log `uri` using the same `redactMongoURI(cfg.MongoURI)` instead of duplicating the redaction logic.
  - Logged fields:
    - `uri`: redacted URI with password removed.
    - `database`: unchanged.
  - Actual connection still uses the full `cfg.MongoURI`; only logs are sanitized.
> Ensures MongoDB credentials never appear in application logs; redaction logic is defined once and reused.

## DNS-01 certificate renewal logic (improved)

- **File:** `internal/app/system/server/dnslego.go`
- Enhanced `obtainOrLoadDNSCert(cfg)` to safely reuse, validate, and renew cached DNS-01 certificates.
- Added renewal window:  
  `renewBefore = 30 * 24 * time.Hour` (30 days)
- Cached cert is reused only if:
  - It loads successfully.
  - It parses correctly.
  - It is **not expired**.
  - It has **more than 30 days remaining** until `NotAfter`.
- Otherwise, a **new certificate** is obtained via DNS-01 (lego + Route53).

### Improved parsing of cached certificates
- Added proper PEM decoding using `encoding/pem`.
- Extracts the first `CERTIFICATE` PEM block.
- Parses using `x509.ParseCertificate`.
- Fixes earlier limitation where cached certs were never reused because `x509.ParseCertificates` expected DER, not PEM.

### Behavior summary
- **On startup:**
  - Valid cached cert → **reused**
  - Expired / invalid / near-expiry → **renewed automatically**
- DNS-01 mode continues to **not require port 80**, allowing HTTPS on non-standard ports (e.g., 8443).
- **No periodic renewal yet:**
  - Renewal occurs **at startup only**.
  - *(Future enhancement)* Full auto-renew for long-running uptime would involve:
    - Switching to `GetCertificate` callback
    - Running a background renewal goroutine

> **For strata_hub:** replicate this DNS-01 renewal logic exactly when adding Route53-based certificate management.

