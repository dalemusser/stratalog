// cmd/stratalog/main.go
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/dalemusser/stratalog/internal/app/system/config"
	"github.com/dalemusser/stratalog/internal/app/system/indexes"
	"github.com/dalemusser/stratalog/internal/app/system/metrics"
	"github.com/dalemusser/stratalog/internal/app/system/server"
	"github.com/dalemusser/stratalog/internal/app/system/validators"
	"github.com/dalemusser/stratalog/internal/platform/db"
	"github.com/dalemusser/stratalog/internal/platform/render"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func buildLogger(level string, prod bool) (*zap.Logger, error) {
	var cfg zap.Config
	if prod {
		cfg = zap.NewProductionConfig()
		cfg.Encoding = "json"
	} else {
		cfg = zap.NewDevelopmentConfig()
	}
	// Honor desired level; default to info on bad input.
	if err := cfg.Level.UnmarshalText([]byte(level)); err != nil {
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	// RFC-3339 timestamps.
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return cfg.Build()
}

func buildBootstrapLogger() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	l, err := cfg.Build()
	if err != nil {
		return zap.NewNop()
	}
	return l
}

// redactMongoURI strips any password from the Mongo URI before logging.
// It preserves the username (if present) but replaces the password with "****".
func redactMongoURI(raw string) string {
	if raw == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	username := u.User.Username()
	if _, hasPwd := u.User.Password(); hasPwd {
		u.User = url.UserPassword(username, "****")
	} else if username != "" {
		u.User = url.User(username)
	} else {
		u.User = nil
	}
	return u.String()
}

func logStartupSummary(cfg *config.Config) {
	sugar := zap.S()

	// TLS mode summary
	tlsMode := "http-only"
	if cfg.UseHTTPS {
		if cfg.UseLetsEncrypt {
			chal := strings.ToLower(strings.TrimSpace(cfg.LetsEncryptChallenge))
			if chal == "" || chal == "http-01" {
				tlsMode = "https (Let's Encrypt http-01)"
			} else {
				tlsMode = "https (Let's Encrypt dns-01 / Route53)"
			}
		} else {
			tlsMode = "https (manual certs)"
		}
	}

	// Request size summary
	reqLimit := cfg.MaxRequestBodyBytes
	reqLimitHuman := "unlimited"
	if reqLimit > 0 {
		reqLimitHuman = fmt.Sprintf("%d bytes", reqLimit)
	}

	// API key presence (redacted)
	hasIngestKey := strings.TrimSpace(cfg.IngestAPIKey) != ""
	hasAdminKey := strings.TrimSpace(cfg.AdminAPIKey) != ""

	// DB URI (redacted: strip password if present)
	redactedMongoURI := redactMongoURI(cfg.MongoURI)

	sugar.Infow("startup configuration",
		// runtime
		"env", cfg.Env,
		"log_level", cfg.LogLevel,

		// network / TLS
		"http_port", cfg.HTTPPort,
		"https_port", cfg.HTTPSPort,
		"use_https", cfg.UseHTTPS,
		"use_lets_encrypt", cfg.UseLetsEncrypt,
		"tls_mode", tlsMode,
		"domain", cfg.Domain,

		// ACME details (non-secret)
		"lets_encrypt_challenge", cfg.LetsEncryptChallenge,
		"lets_encrypt_cache_dir", cfg.LetsEncryptCacheDir,

		// DB (URI redacted; no credentials logged)
		"mongo_uri", redactedMongoURI,
		"mongo_database", cfg.MongoDatabase,
		"index_boot_timeout", cfg.IndexBootTimeout.String(),

		// HTTP behavior
		"max_request_body_bytes", reqLimit,
		"max_request_body_human", reqLimitHuman,
		"enable_compression", cfg.EnableCompression,
		"enable_cors", cfg.EnableCORS,

		// CORS footprint
		"cors_allowed_origins", len(cfg.CORSAllowedOrigins),
		"cors_allowed_methods", len(cfg.CORSAllowedMethods),
		"cors_allowed_headers", len(cfg.CORSAllowedHeaders),

		// security (presence only, values redacted)
		"has_ingest_api_key", hasIngestKey,
		"has_admin_api_key", hasAdminKey,

		// observability
		"metrics_endpoint", "/metrics",
		"pprof_prefix", "/debug/pprof",
		"health_endpoint", "/health",
	)
}

func main() {
	// -------------------------------------------------------------------
	// Step 1: Bootstrap logger so early failures are visible and
	//         register Prometheus metrics
	// -------------------------------------------------------------------
	bootstrap := buildBootstrapLogger()
	zap.ReplaceGlobals(bootstrap)
	zap.L().Info("Step 1 bootstrap logger initialized", zap.String("encoding", "console"), zap.String("level", "info"))

	metrics.RegisterDefaultPrometheus()
	metrics.MustRegisterMetrics()

	// -------------------------------------------------------------------
	// Step 2: Load config
	// -------------------------------------------------------------------
	zap.L().Info("Step 2: loading config…")
	cfg, err := config.Load()
	if err != nil {
		zap.L().Fatal("config load failed", zap.Error(err))
	}
	zap.L().Info("Step 2 complete: config loaded", zap.String("env", cfg.Env), zap.String("log_level", cfg.LogLevel))
	zap.L().Debug("effective config (redacted)", zap.String("config", cfg.Dump()))

	// -------------------------------------------------------------------
	// Step 3: Build final logger
	// -------------------------------------------------------------------
	zap.L().Info("Step 3: building logger…")
	logger, err := buildLogger(cfg.LogLevel, cfg.Env == "prod")
	if err != nil {
		zap.L().Fatal("logger build failed", zap.Error(err))
	}
	defer func() { _ = logger.Sync() }()
	zap.ReplaceGlobals(logger)
	sugar := zap.S()
	sugar.Infow("Step 3 complete: logger initialized", "env", cfg.Env, "level", cfg.LogLevel)

	// Log one-shot startup summary (non-secret)
	logStartupSummary(cfg)

	// -------------------------------------------------------------------
	// Step 4: Connect to MongoDB
	// -------------------------------------------------------------------
	redactedMongoURI := redactMongoURI(cfg.MongoURI)

	sugar.Infow("Step 4: connecting to MongoDB…",
		"uri", redactedMongoURI,
		"database", cfg.MongoDatabase,
	)

	client, err := db.Connect(cfg.MongoURI, cfg.MongoDatabase)
	if err != nil {
		sugar.Fatalw("MongoDB connection failed", "error", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(ctx)
	}()
	sugar.Infow("Step 4 complete: MongoDB connected", "database", cfg.MongoDatabase)

	// -------------------------------------------------------------------
	// Step 5: Ensure collections & validators (idempotent; skip if unsupported)
	// -------------------------------------------------------------------
	sugar.Infow("Step 5: ensuring collections & validators…", "database", cfg.MongoDatabase)
	{
		bootCtx, cancel := context.WithTimeout(context.Background(), cfg.IndexBootTimeout)
		defer cancel()

		database := client.Database(cfg.MongoDatabase)
		if err := validators.EnsureAll(bootCtx, database); err != nil {
			// This returns an error only for real failures; engines that
			// don't support validators are handled inside EnsureAll.
			sugar.Fatalw("ensuring collections/validators failed", "error", err)
		}
	}
	sugar.Infow("Step 5 complete: collections & validators ensured")

	// -------------------------------------------------------------------
	// Step 6: Ensure MongoDB indexes (idempotent; fail fast if broken)
	// -------------------------------------------------------------------
	sugar.Infow("Step 6: ensuring MongoDB indexes…", "database", cfg.MongoDatabase)
	{
		bootCtx, cancel := context.WithTimeout(context.Background(), cfg.IndexBootTimeout)
		defer cancel()

		database := client.Database(cfg.MongoDatabase)
		if err := indexes.EnsureAll(bootCtx, database); err != nil {
			sugar.Fatalw("ensuring MongoDB indexes failed", "error", err)
		}
	}
	sugar.Infow("Step 6 complete: indexes ensured")

	// -------------------------------------------------------------------
	// Step 7: Boot template engine (must be before starting the server)
	// -------------------------------------------------------------------
	// devMode: true in non-prod so you can add hot-refresh later if you want
	eng := render.New(cfg.Env != "prod")

	if err := eng.Boot(); err != nil {
		sugar.Fatalw("Step 7 template engine boot failed", "error", err)
	}
	render.UseEngine(eng)
	sugar.Infow("Step 7 template engine ready")

	// -------------------------------------------------------------------
	// Step 8: Wire shutdown signals → context
	// -------------------------------------------------------------------
	sugar.Infow("Step 8 wiring shutdown signals")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	// Always listen for Ctrl+C (os.Interrupt). Add SIGTERM on non-Windows.
	if runtime.GOOS == "windows" {
		signal.Notify(sigCh, os.Interrupt)
	} else {
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	}
	go func() {
		sig := <-sigCh
		zap.L().Info("shutdown signal received", zap.Any("signal", sig))
		cancel()
	}()

	// -------------------------------------------------------------------
	// Step 9: Start HTTP server (context-cancellable)
	// -------------------------------------------------------------------
	sugar.Infow("Step 9: starting HTTP server…")
	if err := server.StartServerWithContext(ctx, cfg, client); err != nil {
		sugar.Fatalw("server exited with error", "error", err)
	}
	sugar.Infow("server stopped")
}
