1. What’s the problem with server.StartServerWithContext mixing concerns?

Right now StartServerWithContext is doing a lot:
	1.	Router creation
r := chi.NewRouter()
	2.	Middleware wiring
r.Use(...) for RequestID, RealIP, recoverer, body limit, metrics, CORS, compression, logger, etc.
	3.	Route mounting
routes.RegisterAllRoutes(r, h)
	4.	TLS mode selection
Switch between:
	•	HTTP only
	•	HTTPS + LE http-01
	•	HTTPS + LE dns-01
	•	HTTPS + manual cert
	5.	Listener binding
net.Listen(...), wrapping with tls.NewListener(...), etc.
	6.	Graceful shutdown & error loop
The for { select { ... } } over ctx.Done(), serveErr, auxErr.

All of those are legitimate concerns; they just don’t have to live in one function.

The downsides:
	•	Harder to test:
	•	You can’t test “just the router wiring” or “just the TLS selection” without dragging everything in.
	•	Harder to reuse across services:
	•	stratalog, stratasave, strata_hub all want almost the same “server spine” — but with this monolith, you copy/paste or mentally sync them.
	•	Harder to read:
	•	As the number of middlewares/routes grows, the top of StartServerWithContext gets busy.
	•	Harder to evolve:
	•	Adding HTTP/2-only, h2c, or listening on multiple addresses becomes more awkward.

A cleaner structure is:
	•	“Something builds a fully-configured router.”
	•	“Something takes that router + config and runs it with HTTP/TLS + graceful shutdown.”

Even if they’re in the same file/package, that separation makes life nicer.

⸻

2. What would a split server look like?

Imagine two key functions:

A) Router builder

```
func NewRouter(cfg *config.Config, h *handler.Handler, logger *zap.Logger) http.Handler {
    r := chi.NewRouter()

    r.NotFound(httputil.NotFoundHandler(logger))
    r.MethodNotAllowed(httputil.MethodNotAllowedHandler(logger))

    r.Use(blockScans)
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(zapRecoverer(logger))
    r.Use(httputil.LimitBodySize(cfg.HTTP.MaxRequestBodyBytes))
    r.Use(appmetrics.Metrics)
    if cfg.CORS.Enable {
        r.Use(corsMiddleware(&cfg.CORS))
    }
    if cfg.HTTP.EnableCompression {
        r.Use(compressionMiddleware())
    }
    r.Use(zapRequestLogger(logger))

    routes.RegisterAllRoutes(r, h)
    return r
}
```

This function’s job:
“Given config + handler + logger, return a fully wired router.”
No TLS, no listening, no shutdown. Very easy to test with httptest.NewServer.

B) Server runner

```
func ListenAndServeWithContext(ctx context.Context, cfg *config.Config, handler http.Handler, logger *zap.Logger) error {
    srv := &http.Server{
        Handler:           handler,
        ReadTimeout:       15 * time.Second,
        ReadHeaderTimeout: 10 * time.Second,
        WriteTimeout:      60 * time.Second,
        IdleTimeout:       120 * time.Second,
    }

    // wire srv.ErrorLog from logger

    // switch on cfg.TLS/HTTP to bind httpAddr/httpsAddr (all your existing TLS logic)
    // start servePrimary and aux server
    // graceful shutdown select { case <-ctx.Done(): ... }

    return nil
}
```

This function’s job:
“Given config + HTTP handler + logger, choose HTTP/TLS mode, bind sockets, run, and shut down gracefully.”

Now main can do:

```
h := handler.NewHandler(cfg, mongoClient, logger)
router := server.NewRouter(cfg, h, logger)
if err := server.ListenAndServeWithContext(ctx, cfg, router, logger); err != nil {
    sugar.Fatalw("server exited with error", "error", err)
}
```
This is basically what you have now, but split along a logical seam.

⸻

3. How does this tie into a shared strata_core / gowebcore?

You have a recurring pattern across your services:
	•	Config: HTTP/TLS/DB/Logging/Security/CORS.
	•	Core server spine: chi router + common middlewares + TLS selection + graceful shutdown.
	•	Observability: metrics, pprof, health, version.
	•	Utility: httputil, logging helpers, redaction, etc.

A strata_core (or gowebcore) could offer:
	•	core/config: shared config struct + loader.
	•	core/server:
	•	NewRouter that accepts:
	•	a function to mount app-specific routes,
	•	and maybe app-specific middleware.
	•	ListenAndServeWithContext, with the TLS and shutdown logic generalized.
	•	core/observability:
	•	metrics registration (RegisterDefaultPrometheus, MustRegisterMetrics),
	•	pprof mounting,
	•	HTTP histogram middleware.
	•	core/httputil:
	•	JSON helpers, NotFound/MethodNotAllowed, RequireJSON, LimitBodySize, etc.
	•	core/logging:
	•	buildLogger, buildBootstrapLogger,
	•	maybe a helper to attach zap.Logger to http.Server.ErrorLog.

Then stratalog, stratasave, strata_hub all become thin layers:
	•	Their own cmd/<service>/main.go,
	•	Their own internal/app/features/*,
	•	Maybe some custom middleware/route groups.

Your current stratalog code is already a strong proto of this core. A future refactor is mostly about extracting the “common bits” into a dedicated module and letting each service just hook in its own features/routes.

⸻

4. When to change logging relative to this refactor?

You asked:

Would you change logging (passing to functions) first, during or after these changes?

Best order, if we’re being strategic:

Step 1: Logger injection (*zap.Logger instead of zap.L())
	•	Change StartServerWithContext → StartServerWithContext(ctx, cfg, client, logger).
	•	Change NewHandler → accept logger.
	•	Update:
	•	zapRecoverer(logger)
	•	zapRequestLogger(logger)
	•	httputil.NotFoundHandler(logger)
	•	httputil.MethodNotAllowedHandler(logger)
	•	dnslego.go (optional – can stay on zap.L() if you want, since it’s lower-level).

Why first?
	•	Once everything uses an injected logger, splitting server into NewRouter / ListenAndServe is much easier, because they can both take the same logger parameter.
	•	It also immediately improves testability, independent of any further refactor.

Step 2: Split server into NewRouter + ListenAndServeWithContext

After logger injection, you can:
	•	Move the router-building part of StartServerWithContext into NewRouter.
	•	Move the TLS/listener/shutdown portion into ListenAndServeWithContext.
	•	main now owns the glue:

```
h := handler.NewHandler(cfg, client, logger)
router := server.NewRouter(cfg, h, logger)
if err := server.ListenAndServeWithContext(ctx, cfg, router, logger); err != nil {
    sugar.Fatalw("server exited with error", "error", err)
}
```

Now your server core is cleanly factored within stratalog.

Step 3: Extract a shared core (optional, later)

Once stratalog feels “done” and you’re ready to bring stratasave / strata_hub into the same world:
	•	Create a new module (e.g. github.com/dalemusser/gowebcore).
	•	Copy the refined pieces: config, server, observability, httputil, logging.
	•	Slowly migrate services to use that module instead of local copies.

⸻

TL;DR answer
	•	What’s wrong with server.StartServerWithContext?
It’s doing everything: building the router, wiring middleware, mounting routes, choosing TLS mode, binding sockets, and handling shutdown. It works, but it makes testing, reuse, and mental model harder.
	•	What would it look like after?
Two main functions:
	•	NewRouter(cfg, handler, logger) http.Handler
	•	ListenAndServeWithContext(ctx, cfg, handler, logger) error
plus a potentially extracted shared core module for cross-service reuse.
	•	When to do logger injection?
	•	First: Inject *zap.Logger instead of using zap.L(); this improves testability and sets you up.
	•	Then: Split router vs server runner.
	•	Later: Extract reusable core into a separate module used by all Strata services.

None of this is required to ship stratalog, but it’s the path from “good service skeleton” to “excellent multi-service core” if you decide to invest in it.


