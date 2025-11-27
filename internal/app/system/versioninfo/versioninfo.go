// internal/app/system/versioninfo/versioninfo.go
package versioninfo

// These are intended to be overridden at build time via -ldflags, e.g.:
//
//	go build -ldflags "\
//	  -X github.com/dalemusser/stratalog/internal/app/system/versioninfo.Version=v1.2.3 \
//	  -X github.com/dalemusser/stratalog/internal/app/system/versioninfo.GitCommit=abcdef123 \
//	  -X github.com/dalemusser/stratalog/internal/app/system/versioninfo.BuildTime=2025-11-25T21:37:00Z" \
//	-o stratalog ./cmd/stratalog
//
// In dev builds, they'll just have their default values.
var (
	Service   = "stratalog"
	Version   = "dev"
	GitCommit = ""
	BuildTime = ""
)
